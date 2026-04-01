package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/OpenNSW/nsw/internal/payments"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPaymentService is a mock implementation of the payments.PaymentService interface
type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) CreateCheckoutSession(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payments.CreateCheckoutResponse), args.Error(1)
}

func (m *MockPaymentService) ValidateReference(ctx context.Context, req payments.ValidateReferenceRequest) (*payments.ValidateReferenceResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payments.ValidateReferenceResponse), args.Error(1)
}

func (m *MockPaymentService) ProcessWebhook(ctx context.Context, payload payments.WebhookPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

// ── FSM Tests ─────────────────────────────────────────────────────────────────

func TestNewPaymentFSM(t *testing.T) {
	fsm := NewPaymentFSM()

	tests := []struct {
		name      string
		from      string
		action    string
		wantState string
		wantTask  State
		wantOK    bool
	}{
		{"START from empty", "", FSMActionStart, "IDLE", "", true},
		{"INITIATE from IDLE", "IDLE", PaymentActionInitiate, "IN_PROGRESS", InProgress, true},
		{"SUCCESS from IN_PROGRESS", "IN_PROGRESS", PaymentActionSuccess, "COMPLETED", Completed, true},
		{"FAILED from IN_PROGRESS", "IN_PROGRESS", PaymentActionFailed, "IDLE", Initialized, true},
		{"TIMEOUT from IN_PROGRESS", "IN_PROGRESS", paymentFSMTimeout, "IDLE", Initialized, true},

		// Invalid transitions
		{"INITIATE from empty", "", PaymentActionInitiate, "", "", false},
		{"SUCCESS from IDLE", "IDLE", PaymentActionSuccess, "", "", false},
		{"FAILED from IDLE", "IDLE", PaymentActionFailed, "", "", false},
		{"INITIATE from COMPLETED", "COMPLETED", PaymentActionInitiate, "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := fsm.CanTransition(tt.from, tt.action)
			assert.Equal(t, tt.wantOK, ok)

			if tt.wantOK {
				outcome, err := fsm.Transition(tt.from, tt.action)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, outcome.NextPluginState)
				assert.Equal(t, tt.wantTask, outcome.NextTaskState)
			}
		})
	}
}

// ── Constructor Tests ─────────────────────────────────────────────────────────

func TestNewPaymentTask(t *testing.T) {
	mockSvc := new(MockPaymentService)
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := `{"currency": "USD", "ttl": 300, "orgId": "CUSTOMS", "breakdown": [{"description": "Fee", "category": "ADDITION", "type": "FIXED", "unitPrice": "100.50"}]}`
		task, err := NewPaymentTask(json.RawMessage(cfg), mockSvc)
		assert.NoError(t, err)
		assert.Equal(t, "USD", task.config.Currency)
		assert.Equal(t, 300, task.config.TTL)
		assert.Equal(t, "CUSTOMS", task.config.OrgID)
		assert.Len(t, task.config.Breakdown, 1)
		assert.Equal(t, "100.50", task.config.Breakdown[0].UnitPrice)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := NewPaymentTask(json.RawMessage(`{invalid`), mockSvc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config")
	})
}

// ── Start Tests ───────────────────────────────────────────────────────────────

func TestPaymentStart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()
		mockAPI.On("Transition", FSMActionStart).Return(nil).Once()

		resp, err := task.Start(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Payment task started", resp.Message)

		// Verify session was written with a valid transaction ID.
		call := mockAPI.Calls[1] // WriteToLocalStore call
		session := call.Arguments[1].(*PaymentSession)
		assert.NotEmpty(t, session.TransactionID)
		assert.False(t, session.GeneratedAt.IsZero())
		assert.Nil(t, session.InitiatedAt)

		mockAPI.AssertExpectations(t)
	})

	t.Run("AlreadyStarted", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(false).Once()

		resp, err := task.Start(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "Payment task already started", resp.Message)

		mockAPI.AssertExpectations(t)
	})

	t.Run("PersistInitialSessionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(errors.New("store failed")).Once()

		resp, err := task.Start(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist initial session")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()
		mockAPI.On("Transition", FSMActionStart).Return(errors.New("transition failed")).Once()

		resp, err := task.Start(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition failed")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}

// ── GetRenderInfo Tests ───────────────────────────────────────────────────────

func TestPaymentGetRenderInfo_IDLE(t *testing.T) {
	mockAPI := new(MockAPI)
	mockSvc := new(MockPaymentService)
	task := newTestPaymentTask(mockSvc)
	task.Init(mockAPI)

	session := PaymentSession{
		TransactionID: "txn-123",
		GeneratedAt:   time.Now(), // fresh session, within TTL
	}

	mockAPI.On("GetPluginState").Return("IDLE")
	mockAPI.On("GetTaskState").Return(InProgress)
	mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil)

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	assert.Equal(t, TaskTypePayment, data.Type)
	assert.Equal(t, "IDLE", data.PluginState)

	content := data.Content.(PaymentRenderContent)
	assert.True(t, decimal.NewFromFloat(100.0).Equal(content.TotalAmount))
	assert.Equal(t, "USD", content.Currency)

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_Completed(t *testing.T) {
	mockAPI := new(MockAPI)
	mockSvc := new(MockPaymentService)
	task := newTestPaymentTask(mockSvc)
	task.Init(mockAPI)

	mockAPI.On("GetPluginState").Return("COMPLETED")
	mockAPI.On("GetTaskState").Return(Completed)

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	content := data.Content.(PaymentRenderContent)
	assert.True(t, decimal.NewFromFloat(100.0).Equal(content.TotalAmount))
	assert.Equal(t, "USD", content.Currency)
	assert.Equal(t, "COMPLETED", data.PluginState)

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_SessionRotation(t *testing.T) {
	mockAPI := new(MockAPI)
	mockSvc := new(MockPaymentService)
	task := newTestPaymentTask(mockSvc)
	task.Init(mockAPI)

	// Session generated 10 minutes ago — well past the 5-minute TTL.
	expiredSession := PaymentSession{
		TransactionID: "old-txn",
		GeneratedAt:   time.Now().Add(-10 * time.Minute),
	}

	mockAPI.On("GetPluginState").Return("IDLE")
	mockAPI.On("GetTaskState").Return(InProgress)
	mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(expiredSession, nil)
	mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	content := data.Content.(PaymentRenderContent)
	// The rotated session should have a new transaction ID, not the old one.
	assert.NotContains(t, content.ReferenceNumber, "old-txn")

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_TimeoutTransition(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		// Session initiated 10 minutes ago — well past TTL (5m) + Threshold (30s).
		initiatedAt := time.Now().Add(-10 * time.Minute)
		expiredSession := PaymentSession{
			TransactionID: "stale-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		// First call returns IN_PROGRESS, second (after timeout transition) returns IDLE.
		mockAPI.On("GetPluginState").Return("IN_PROGRESS").Once()
		mockAPI.On("GetPluginState").Return("IDLE").Once()
		mockAPI.On("GetTaskState").Return(InProgress)
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(expiredSession, nil).Once()

		// Expect timeout transaction recording.
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).Return(nil).Once()
		mockAPI.On("Transition", paymentFSMTimeout).Return(nil).Once()

		// Expect session rotation (session is also past TTL).
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()

		resp, err := task.GetRenderInfo(context.Background())

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(GetRenderInfoResponse)
		assert.Equal(t, "IDLE", data.PluginState)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-10 * time.Minute)
		expiredSession := PaymentSession{
			TransactionID: "stale-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("GetPluginState").Return("IN_PROGRESS").Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(expiredSession, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).Return(nil).Once()
		mockAPI.On("Transition", paymentFSMTimeout).Return(errors.New("transition failed")).Once()

		resp, err := task.GetRenderInfo(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed timeout transition")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("RecordTimeoutError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-10 * time.Minute)
		expiredSession := PaymentSession{
			TransactionID: "stale-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("GetPluginState").Return("IN_PROGRESS").Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(expiredSession, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, errors.New("read history failed")).Once()

		resp, err := task.GetRenderInfo(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record timeout transaction")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}

// ── Execute Tests ─────────────────────────────────────────────────────────────

func TestPaymentExecute_InitiatePayment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		session := PaymentSession{
			TransactionID: "txn-456",
			GeneratedAt:   time.Now(), // fresh session
		}

		mockAPI.On("CanTransition", PaymentActionInitiate).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()
		mockAPI.On("GetTaskID").Return("task-123").Once()
		mockAPI.On("GetPluginState").Return("IDLE").Once()

		mockSvc.On("CreateCheckoutSession", mock.Anything, mock.MatchedBy(func(req payments.CreateCheckoutRequest) bool {
			return req.Amount.Equal(decimal.NewFromFloat(100.0)) && req.Currency == "USD"
		})).Return(&payments.CreateCheckoutResponse{
			SessionID:   "sess-123",
			CheckoutURL: "https://pay.example.com/sess-123",
		}, nil).Once()

		var capturedSession *PaymentSession
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).
			Run(func(args mock.Arguments) {
				capturedSession = args.Get(1).(*PaymentSession)
			}).Return(nil).Once()
		mockAPI.On("Transition", PaymentActionInitiate).Return(nil).Once()

		initiatedAt := time.Now().Format(time.RFC3339)
		req := &ExecutionRequest{
			Action:  PaymentActionInitiate,
			Content: map[string]any{"initiatedAt": initiatedAt},
		}

		resp, err := task.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Payment initiated", resp.Message)
		assert.True(t, resp.ApiResponse.Success)

		// Verify InitiatedAt was set on the session.
		assert.NotNil(t, capturedSession.InitiatedAt)
		assert.Equal(t, "https://pay.example.com/sess-123", capturedSession.CheckoutURL)

		mockAPI.AssertExpectations(t)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ExpiredSession", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		expiredSession := PaymentSession{
			TransactionID: "old-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute), // well past 5m TTL
		}

		mockAPI.On("CanTransition", PaymentActionInitiate).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(expiredSession, nil).Once()

		req := &ExecutionRequest{
			Action:  PaymentActionInitiate,
			Content: map[string]any{},
		}

		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session expired")
		assert.NotNil(t, resp)
		assert.False(t, resp.ApiResponse.Success)
		assert.Equal(t, "SESSION_EXPIRED", resp.ApiResponse.Error.Code)

		mockAPI.AssertExpectations(t)
	})

	t.Run("InvalidTransition", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionInitiate).Return(false).Once()
		mockAPI.On("GetPluginState").Return("COMPLETED")

		req := &ExecutionRequest{
			Action:  PaymentActionInitiate,
			Content: map[string]any{},
		}

		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}

func TestPaymentExecute_PaymentSuccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionSuccess).Return(true).Once()
		mockAPI.On("Transition", PaymentActionSuccess).Return(nil).Once()

		req := &ExecutionRequest{Action: PaymentActionSuccess}
		resp, err := task.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Payment completed successfully", resp.Message)
		assert.True(t, resp.ApiResponse.Success)

		mockAPI.AssertExpectations(t)
	})

	t.Run("InvalidTransition", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionSuccess).Return(false).Once()
		mockAPI.On("GetPluginState").Return("IDLE").Once()

		req := &ExecutionRequest{Action: PaymentActionSuccess}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionSuccess).Return(true).Once()
		mockAPI.On("Transition", PaymentActionSuccess).Return(errors.New("transition failed")).Once()

		req := &ExecutionRequest{Action: PaymentActionSuccess}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition failed")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}

func TestPaymentExecute_PaymentFailed(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-2 * time.Minute)
		session := PaymentSession{
			TransactionID: "txn-789",
			GeneratedAt:   time.Now().Add(-3 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()

		// Expect transaction history read (empty) and write.
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		var capturedTxns []PaymentTransaction
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).
			Run(func(args mock.Arguments) {
				capturedTxns = args.Get(1).([]PaymentTransaction)
			}).Return(nil).Once()

		// Expect new session write.
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()
		mockAPI.On("Transition", PaymentActionFailed).Return(nil).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "failed")
		assert.True(t, resp.ApiResponse.Success)

		// Verify the transaction was recorded.
		assert.Len(t, capturedTxns, 1)
		assert.Equal(t, "txn-789", capturedTxns[0].TransactionID)
		assert.Equal(t, "FAILED", capturedTxns[0].Status)
		assert.Equal(t, 1, capturedTxns[0].Round)

		mockAPI.AssertExpectations(t)
	})

	t.Run("ReadHistoryError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-2 * time.Minute)
		session := PaymentSession{
			TransactionID: "txn-789",
			GeneratedAt:   time.Now().Add(-3 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, errors.New("read history failed")).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record failed transaction")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("PersistHistoryError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-2 * time.Minute)
		session := PaymentSession{
			TransactionID: "txn-789",
			GeneratedAt:   time.Now().Add(-3 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).Return(errors.New("write history failed")).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record failed transaction")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("PersistNewSessionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-2 * time.Minute)
		session := PaymentSession{
			TransactionID: "txn-789",
			GeneratedAt:   time.Now().Add(-3 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).Return(nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(errors.New("persist session failed")).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist new session")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-2 * time.Minute)
		session := PaymentSession{
			TransactionID: "txn-789",
			GeneratedAt:   time.Now().Add(-3 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(session, nil).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreTransactions, mock.AnythingOfType("[]plugin.PaymentTransaction")).Return(nil).Once()
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()
		mockAPI.On("Transition", PaymentActionFailed).Return(errors.New("transition failed")).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition failed")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}

func TestPaymentHelpers_readSession(t *testing.T) {
	t.Run("ReadError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(nil, errors.New("read failed")).Once()

		result, err := task.readSession(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("NilSession", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(nil, nil).Once()

		result, err := task.readSession(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active payment session")
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathSuccess", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		now := time.Now().UTC().Format(time.RFC3339)
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(map[string]any{
			"transactionId": "txn-1",
			"generatedAt":   now,
			"initiatedAt":   now,
		}, nil).Once()

		result, err := task.readSession(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "txn-1", result.TransactionID)
		assert.NotNil(t, result.InitiatedAt)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathMarshalError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(map[string]any{"bad": func() {}}, nil).Once()

		result, err := task.readSession(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal stored session")
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathUnmarshalError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(map[string]any{
			"transactionId": "txn-1",
			"generatedAt":   "not-a-time",
		}, nil).Once()

		result, err := task.readSession(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal stored session")
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})
}

func TestPaymentHelpers_readTransactionHistory(t *testing.T) {
	t.Run("ReadError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(nil, errors.New("read failed")).Once()

		result, err := task.readTransactionHistory(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathSuccess", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		now := time.Now().UTC().Format(time.RFC3339)
		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return([]any{
			map[string]any{
				"transactionId": "txn-1",
				"initiatedAt":   now,
				"resolvedAt":    now,
				"status":        "FAILED",
				"round":         1,
			},
		}, nil).Once()

		result, err := task.readTransactionHistory(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "txn-1", result[0].TransactionID)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathMarshalError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return(map[string]any{"bad": func() {}}, nil).Once()

		result, err := task.readTransactionHistory(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal stored transactions")
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("SlowPathUnmarshalError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		mockSvc := new(MockPaymentService)
		task := newTestPaymentTask(mockSvc)
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreTransactions).Return([]any{
			map[string]any{
				"transactionId": "txn-1",
				"initiatedAt":   "bad-time",
				"resolvedAt":    "bad-time",
				"status":        "FAILED",
				"round":         1,
			},
		}, nil).Once()

		result, err := task.readTransactionHistory(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal stored transactions")
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})
}

func TestPaymentExecute_NilRequest(t *testing.T) {
	mockAPI := new(MockAPI)
	mockSvc := new(MockPaymentService)
	task := newTestPaymentTask(mockSvc)
	task.Init(mockAPI)

	resp, err := task.Execute(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request is required")
	assert.Nil(t, resp)
}

func TestPaymentExecute_UnknownAction(t *testing.T) {
	mockAPI := new(MockAPI)
	mockSvc := new(MockPaymentService)
	task := newTestPaymentTask(mockSvc)
	task.Init(mockAPI)

	req := &ExecutionRequest{Action: "UNKNOWN"}
	resp, err := task.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
	assert.Nil(t, resp)
}

// ── Helper ────────────────────────────────────────────────────────────────────

// newTestPaymentTask creates a PaymentTask with a standard test configuration.
func newTestPaymentTask(mockSvc payments.PaymentService) *PaymentTask {
	return &PaymentTask{
		config: PaymentConfig{
			Currency: "USD",
			TTL:      300, // 5 minutes
			OrgID:    "CUSTOMS",
			Breakdown: []BreakdownItem{
				{
					Description: "Test Fee",
					Category:    CategoryAddition,
					Type:        TypeFixed,
					UnitPrice:   "100.0",
				},
			},
		},
		paymentService: mockSvc,
	}
}

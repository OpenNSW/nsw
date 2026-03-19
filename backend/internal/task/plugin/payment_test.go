package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

type testPaymentRepo struct {
	db *gorm.DB
}

func (r *testPaymentRepo) CreateTransaction(ctx context.Context, trx *payment_types.PaymentTransactionDB) error {
	return r.db.WithContext(ctx).Create(trx).Error
}

func (r *testPaymentRepo) GetTransactionByReference(ctx context.Context, ref string, forUpdate bool) (*payment_types.PaymentTransactionDB, error) {
	var trx payment_types.PaymentTransactionDB
	err := r.db.WithContext(ctx).Where("reference_number = ?", ref).First(&trx).Error
	return &trx, err
}

func (r *testPaymentRepo) GetTransactionByExecutionID(ctx context.Context, execID string) (*payment_types.PaymentTransactionDB, error) {
	var trx payment_types.PaymentTransactionDB
	err := r.db.WithContext(ctx).Where("execution_id = ?", execID).First(&trx).Error
	return &trx, err
}

func (r *testPaymentRepo) UpdateTransactionStatus(ctx context.Context, ref string, status string) error {
	return r.db.WithContext(ctx).Model(&payment_types.PaymentTransactionDB{}).
		Where("reference_number = ?", ref).
		Update("status", status).Error
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
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := `{"amount": 100.50, "currency": "USD", "gatewayId": "mock", "ttl": 300}`
		task, err := NewPaymentTask(json.RawMessage(cfg), &config.Config{}, nil, &gateway.MockGateway{})
		assert.NoError(t, err)
		assert.Equal(t, 100.50, task.config.Amount)
		assert.Equal(t, "USD", task.config.Currency)
		assert.Equal(t, "mock", task.config.GatewayID)
		assert.Equal(t, 300, task.config.TTL)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := NewPaymentTask(json.RawMessage(`{invalid`), nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config")
	})
}

// ── Start Tests ───────────────────────────────────────────────────────────────

func TestPaymentStart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()

		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		var capturedSession *PaymentSession
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).
			Run(func(args mock.Arguments) {
				capturedSession = args.Get(1).(*PaymentSession)
			}).Return(nil).Once()
		mockAPI.On("Transition", FSMActionStart).Return(nil).Once()

		resp, err := task.Start(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Payment task started", resp.Message)

		// Verify session was written with a valid transaction ID.
		assert.NotNil(t, capturedSession)
		assert.NotEmpty(t, capturedSession.TransactionID)
		assert.False(t, capturedSession.GeneratedAt.IsZero())
		assert.Nil(t, capturedSession.InitiatedAt)

		mockAPI.AssertExpectations(t)
	})

	t.Run("AlreadyStarted", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(false).Once()

		resp, err := task.Start(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "Payment task already started", resp.Message)

		mockAPI.AssertExpectations(t)
	})

	t.Run("PersistInitialSessionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()

		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(errors.New("store failed")).Once()

		resp, err := task.Start(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist initial session")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", FSMActionStart).Return(true).Once()

		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

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
	task, dbMock := newTestPaymentTask()
	task.gateway = nil // Add this line to test the fallback URL
	task.Init(mockAPI)

	session := PaymentSession{
		TransactionID: "txn-123",
		GeneratedAt:   time.Now(), // fresh session, within TTL
	}

	mockAPI.On("GetPluginState").Return("IDLE")
	mockAPI.On("GetTaskState").Return(InProgress)
	mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&session, nil)

	// Expect query for GetGatewayURL (via GetRenderInfo)
	dbMock.ExpectQuery("SELECT \\* FROM \"payment_transactions\"").
		WithArgs(session.TransactionID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"reference_number", "amount", "currency"}).
			AddRow(session.TransactionID, 100.0, "USD"))

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	assert.Equal(t, TaskTypePayment, data.Type)
	assert.Equal(t, "IDLE", data.PluginState)

	content := data.Content.(PaymentRenderContent)
	// Gateway URL should be fallback or mocked URL
	assert.Contains(t, content.GatewayURL, "txn-123")
	assert.Equal(t, 100.0, content.Amount)
	assert.Equal(t, "USD", content.Currency)

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_Completed(t *testing.T) {
	mockAPI := new(MockAPI)
	task, _ := newTestPaymentTask()
	task.Init(mockAPI)

	mockAPI.On("GetPluginState").Return("COMPLETED")
	mockAPI.On("GetTaskState").Return(Completed)

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	content := data.Content.(PaymentRenderContent)
	assert.Equal(t, 100.0, content.Amount)
	assert.Equal(t, "USD", content.Currency)
	assert.Equal(t, "COMPLETED", data.PluginState)

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_SessionRotation(t *testing.T) {
	mockAPI := new(MockAPI)
	task, dbMock := newTestPaymentTask()
	task.Init(mockAPI)

	// Session generated 10 minutes ago — well past the 5-minute TTL.
	expiredSession := PaymentSession{
		TransactionID: "old-txn",
		GeneratedAt:   time.Now().Add(-10 * time.Minute),
	}

	mockAPI.On("GetPluginState").Return("IDLE")
	mockAPI.On("GetTaskState").Return(InProgress)
	mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&expiredSession, nil)

	// Session rotation calls createTransactionRecord
	mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	// During rotation, GetRenderInfo also fetches the new transaction!
	dbMock.ExpectQuery("SELECT \\* FROM \"payment_transactions\"").
		WillReturnRows(sqlmock.NewRows([]string{"reference_number", "amount", "currency"}).
			AddRow("new-txn-123", 100.0, "USD"))

	mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()

	resp, err := task.GetRenderInfo(context.Background())

	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(GetRenderInfoResponse)
	content := data.Content.(PaymentRenderContent)
	// The rotated session should have a new transaction ID, not the old one.
	assert.NotContains(t, content.GatewayURL, "old-txn")

	mockAPI.AssertExpectations(t)
}

func TestPaymentGetRenderInfo_TimeoutTransition(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
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
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&expiredSession, nil).Once()

		mockAPI.On("Transition", paymentFSMTimeout).Return(nil).Once()

		// Session rotation calls createTransactionRecord
		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		// During timeout rotation, GetRenderInfo also fetches the new transaction!
		dbMock.ExpectQuery("SELECT \\* FROM \"payment_transactions\"").
			WillReturnRows(sqlmock.NewRows([]string{"reference_number", "amount", "currency"}).
				AddRow("new-txn-456", 100.0, "USD"))

		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()

		resp, err := task.GetRenderInfo(context.Background())

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(GetRenderInfoResponse)
		assert.Equal(t, "IDLE", data.PluginState)
		assert.NotContains(t, data.Content.(PaymentRenderContent).GatewayURL, "stale-txn")

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		initiatedAt := time.Now().Add(-10 * time.Minute)
		expiredSession := PaymentSession{
			TransactionID: "stale-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute),
			InitiatedAt:   &initiatedAt,
		}

		mockAPI.On("GetPluginState").Return("IN_PROGRESS").Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&expiredSession, nil).Once()
		mockAPI.On("Transition", paymentFSMTimeout).Return(errors.New("transition failed")).Once()

		resp, err := task.GetRenderInfo(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed timeout transition")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

}

// ── Execute Tests ─────────────────────────────────────────────────────────────

func TestPaymentExecute_InitiatePayment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		session := PaymentSession{
			TransactionID: "txn-456",
			GeneratedAt:   time.Now(), // fresh session
		}

		mockAPI.On("GetPluginState").Return(string(paymentIdle)).Once()
		mockAPI.On("CanTransition", PaymentActionInitiate).Return(true).Once()
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&session, nil).Once()

		// Expect query for transaction lookup during initiation
		dbMock.ExpectQuery("SELECT \\* FROM \"payment_transactions\"").
			WithArgs(session.TransactionID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"reference_number", "task_id", "execution_id", "amount", "currency"}).
				AddRow(session.TransactionID, uuid.New(), uuid.New(), 100.0, "USD"))
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

		mockAPI.AssertExpectations(t)
	})

	t.Run("ExpiredSession", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		expiredSession := PaymentSession{
			TransactionID: "old-txn",
			GeneratedAt:   time.Now().Add(-10 * time.Minute), // well past 5m TTL
		}

		mockAPI.On("GetPluginState").Return(string(paymentIdle)).Once()
		mockAPI.On("CanTransition", PaymentActionInitiate).Return(true).Once()

		// Expect record creation during initiation
		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(&expiredSession, nil).Once()

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
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("GetPluginState").Return(string(paymentInProgress)).Once()
		mockAPI.On("CanTransition", PaymentActionInitiate).Return(false).Once()

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
		task, _ := newTestPaymentTask()
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
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionSuccess).Return(false).Once()

		req := &ExecutionRequest{Action: PaymentActionSuccess}
		resp, err := task.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, _ := newTestPaymentTask()
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
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()

		// Expect new record creation for retry
		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		// Expect new session write.
		mockAPI.On("WriteToLocalStore", paymentStoreSession, mock.AnythingOfType("*plugin.PaymentSession")).Return(nil).Once()
		mockAPI.On("Transition", PaymentActionFailed).Return(nil).Once()

		req := &ExecutionRequest{Action: PaymentActionFailed}
		resp, err := task.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "failed")
		assert.True(t, resp.ApiResponse.Success)

		mockAPI.AssertExpectations(t)
	})

	t.Run("TransitionError", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, dbMock := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("CanTransition", PaymentActionFailed).Return(true).Once()
		mockAPI.On("GetTaskID").Return(uuid.New().String()).Once()
		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO \"payment_transactions\"").WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

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
		task, _ := newTestPaymentTask()
		task.Init(mockAPI)

		mockAPI.On("ReadFromLocalStore", paymentStoreSession).Return(nil, errors.New("read failed")).Once()

		result, err := task.readSession(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
		mockAPI.AssertExpectations(t)
	})

	t.Run("NilSession", func(t *testing.T) {
		mockAPI := new(MockAPI)
		task, _ := newTestPaymentTask()
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
		task, _ := newTestPaymentTask()
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
		task, _ := newTestPaymentTask()
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
		task, _ := newTestPaymentTask()
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

func TestPaymentExecute_NilRequest(t *testing.T) {
	mockAPI := new(MockAPI)
	task, _ := newTestPaymentTask()
	task.Init(mockAPI)

	resp, err := task.Execute(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request is required")
	assert.Nil(t, resp)
}

func TestPaymentExecute_UnknownAction(t *testing.T) {
	mockAPI := new(MockAPI)
	task, _ := newTestPaymentTask()
	task.Init(mockAPI)

	req := &ExecutionRequest{Action: "UNKNOWN"}
	resp, err := task.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
	assert.Nil(t, resp)
}

// ── Helper ────────────────────────────────────────────────────────────────────

// newTestPaymentTask creates a PaymentTask with a standard test configuration and mock DB.
func newTestPaymentTask() (*PaymentTask, sqlmock.Sqlmock) {
	db, mock, _ := sqlmock.New()
	gdb, _ := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	return &PaymentTask{
		config: PaymentConfig{
			Amount:    100.0,
			Currency:  "USD",
			GatewayID: "mock",
			TTL:       300, // 5 minutes
		},
		appConfig: &config.Config{},
		repo:      &testPaymentRepo{db: gdb},
		gateway:   &gateway.MockGateway{},
	}, mock
}

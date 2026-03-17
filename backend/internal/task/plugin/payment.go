package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

// ── Public API Actions ────────────────────────────────────────────────────────

const (
	PaymentActionInitiate = "INITIATE_PAYMENT"
	PaymentActionSuccess  = "PAYMENT_SUCCESS"
	PaymentActionFailed   = "PAYMENT_FAILED"
)

// paymentFSMTimeout is an internal FSM action triggered by the lazy TTL+Threshold
// check. It is not exposed in the public API.
const paymentFSMTimeout = "PAYMENT_TIMEOUT"

// PaymentThreshold is the grace period beyond the TTL before an in-progress
// payment is considered timed out.
const PaymentThreshold = 30 * time.Second

// ── Plugin States ─────────────────────────────────────────────────────────────

type paymentState string

const (
	paymentIdle       paymentState = "IDLE"
	paymentInProgress paymentState = "IN_PROGRESS"
	paymentCompleted  paymentState = "COMPLETED"
)

// ── Local Store Keys ──────────────────────────────────────────────────────────

const (
	paymentStoreSession = "payment:session"
)

// ── Models ─────────────────────────────────────────────────────────────

// PaymentConfig holds the task-level configuration supplied at workflow definition time.
type PaymentConfig struct {
	Amount   float64 `json:"amount"`   // Amount to be paid
	Currency string  `json:"currency"` // Currency of the payment (e.g. "USD")
	Gateway  string  `json:"gateway"`  // Base URL of the payment gateway
	TTL      int     `json:"ttl"`      // Time-to-live for a payment session in seconds
}

// PaymentSession is the current active payment session persisted in local store.
type PaymentSession struct {
	TransactionID string     `json:"transactionId"`
	GeneratedAt   time.Time  `json:"generatedAt"`
	InitiatedAt   *time.Time `json:"initiatedAt,omitempty"` // set when INITIATE_PAYMENT is received
}

// PaymentRenderContent is the payload returned inside GetRenderInfoResponse.Content
type PaymentRenderContent struct {
	GatewayURL string  `json:"gatewayUrl,omitempty"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
}

// ── FSM ───────────────────────────────────────────────────────────────────────

// State graph:
//
//	""            ──START──────────────► IDLE          [no task state change]
//	IDLE          ──INITIATE_PAYMENT──► IN_PROGRESS   [IN_PROGRESS]
//	IN_PROGRESS   ──PAYMENT_SUCCESS───► COMPLETED     [COMPLETED]
//	IN_PROGRESS   ──PAYMENT_FAILED────► IDLE          [INITIALIZED]
//	IN_PROGRESS   ──PAYMENT_TIMEOUT───► IDLE          [INITIALIZED]
//
// NewPaymentFSM returns the state graph for the payment plugin.
func NewPaymentFSM() *PluginFSM {
	return NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:                              {string(paymentIdle), ""},
		{string(paymentIdle), PaymentActionInitiate}:      {string(paymentInProgress), InProgress},
		{string(paymentInProgress), PaymentActionSuccess}: {string(paymentCompleted), Completed},
		{string(paymentInProgress), PaymentActionFailed}:  {string(paymentIdle), Initialized},
		{string(paymentInProgress), paymentFSMTimeout}:    {string(paymentIdle), Initialized},
	})
}

// ── Plugin ────────────────────────────────────────────────────────────────────

// PaymentTask implements Plugin for the PAYMENT task type.
type PaymentTask struct {
	api       API
	config    PaymentConfig
	appConfig *config.Config
	repo      payment_types.PaymentRepository
	gateway   gateway.PaymentGateway
}

// NewPaymentTask creates a PaymentTask from the raw JSON configuration.
func NewPaymentTask(raw json.RawMessage, appCfg *config.Config, repo payment_types.PaymentRepository, gw gateway.PaymentGateway) (*PaymentTask, error) {
	var cfg PaymentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("payment: invalid config: %w", err)
	}
	return &PaymentTask{config: cfg, appConfig: appCfg, repo: repo, gateway: gw}, nil
}

func (t *PaymentTask) Init(api API) {
	t.api = api
}

// ── Start ─────────────────────────────────────────────────────────────────────

func (t *PaymentTask) Start(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(FSMActionStart) {
		return &ExecutionResponse{Message: "Payment task already started"}, nil
	}

	// Persist initial session to local store
	session := PaymentSession{
		TransactionID: uuid.New().String(),
		GeneratedAt:   time.Now(),
	}
	if err := t.api.WriteToLocalStore(paymentStoreSession, &session); err != nil {
		return nil, fmt.Errorf("payment: failed to persist initial session: %w", err)
	}

	if err := t.api.Transition(FSMActionStart); err != nil {
		return nil, err
	}

	return &ExecutionResponse{Message: "Payment task started"}, nil
}

// ── GetRenderInfo ─────────────────────────────────────────────────────────────

func (t *PaymentTask) GetRenderInfo(ctx context.Context) (*ApiResponse, error) {
	pluginState := t.api.GetPluginState()

	if pluginState == string(paymentCompleted) {
		return &ApiResponse{
			Success: true,
			Data: GetRenderInfoResponse{
				Type:        TaskTypePayment,
				PluginState: pluginState,
				State:       t.api.GetTaskState(),
				Content: PaymentRenderContent{
					Amount:   t.config.Amount,
					Currency: t.config.Currency,
				},
			},
		}, nil
	}

	session, err := t.readSession(ctx)
	if err != nil {
		// Resilience: if state is IDLE or empty, returning error is better than partial success
		// as it forces the system to handle the unstarted state.
		return nil, fmt.Errorf("payment: failed to read session: %w", err)
	}

	// Lazy timeout check (if TTL is configured)
	if t.config.TTL > 0 && pluginState == string(paymentInProgress) && session.InitiatedAt != nil {
		deadline := session.InitiatedAt.Add(time.Duration(t.config.TTL)*time.Second + PaymentThreshold)
		if time.Now().After(deadline) {
			if err := t.api.Transition(paymentFSMTimeout); err != nil {
				return nil, fmt.Errorf("payment: failed timeout transition: %w", err)
			}
			pluginState = t.api.GetPluginState()
		}
	}

	// Lazy session rotation (if TTL is configured and state is IDLE or just timed out to IDLE)
	if t.config.TTL > 0 && pluginState == string(paymentIdle) &&
		time.Now().After(session.GeneratedAt.Add(time.Duration(t.config.TTL)*time.Second)) {
		record, err := t.createTransactionRecord(ctx)
		if err != nil {
			return nil, err
		}
		session = &PaymentSession{
			TransactionID: record.ReferenceNumber,
			GeneratedAt:   time.Now(),
		}
		if err := t.api.WriteToLocalStore(paymentStoreSession, session); err != nil {
			return nil, fmt.Errorf("payment: failed to rotate session: %w", err)
		}
	}

	// Generate Gateway URL based on mock mode
	gatewayURL := t.getGatewayURL(session)

	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypePayment,
			PluginState: pluginState,
			State:       t.api.GetTaskState(),
			Content: PaymentRenderContent{
				GatewayURL: gatewayURL,
				Amount:     t.config.Amount,
				Currency:   t.config.Currency,
			},
		},
	}, nil
}

// ── Execute ───────────────────────────────────────────────────────────────────

func (t *PaymentTask) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("payment: execution request is required")
	}

	switch request.Action {
	case PaymentActionInitiate:
		return t.initiateHandler(ctx, request.Content)
	case PaymentActionSuccess:
		return t.successHandler(ctx)
	case PaymentActionFailed:
		return t.failedHandler(ctx)
	default:
		return nil, fmt.Errorf("payment: unknown action %q", request.Action)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────
func (t *PaymentTask) initiateHandler(ctx context.Context, content any) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionInitiate) {
		return nil, fmt.Errorf("payment: action %q not permitted", PaymentActionInitiate)
	}

	session, err := t.readSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to read session: %w", err)
	}

	// Reject if the session has expired
	if t.config.TTL > 0 && time.Now().After(session.GeneratedAt.Add(time.Duration(t.config.TTL)*time.Second)) {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "SESSION_EXPIRED",
					Message: "Payment session has expired. Please restart the payment process.",
				},
			},
		}, fmt.Errorf("payment: session expired")
	}

	// Extract method from content
	var method string
	if m, ok := content.(map[string]any); ok {
		if val, exists := m["method"]; exists {
			method = fmt.Sprintf("%v", val)
		}
	}

	now := time.Now()
	session.InitiatedAt = &now
	if err := t.api.WriteToLocalStore(paymentStoreSession, session); err != nil {
		return nil, err
	}

	// Transition Plugin state: IDLE -> IN_PROGRESS
	// Note: t.api.Transition will check the FSM permission.
	if err := t.api.Transition(PaymentActionInitiate); err != nil {
		return nil, err
	}

	// Instant success for CARD in mock mode
	if method == "CARD" && t.appConfig != nil && t.appConfig.Payment.MockMode {
		if err := t.repo.UpdateTransactionStatus(ctx, session.TransactionID, "COMPLETED"); err != nil {
			return nil, fmt.Errorf("payment: failed to update transaction status: %w", err)
		}

		// Transition Plugin state: IN_PROGRESS -> COMPLETED
		if err := t.api.Transition(PaymentActionSuccess); err != nil {
			return nil, err
		}

		return &ExecutionResponse{
			Message: "Payment completed successfully (Instant Card)",
			ApiResponse: &ApiResponse{
				Success: true,
				Data: map[string]any{
					"message":    "Payment completed successfully",
					"gatewayUrl": "",
				},
			},
		}, nil
	}

	return &ExecutionResponse{
		Message: "Payment initiated",
		ApiResponse: &ApiResponse{
			Success: true,
			Data: map[string]any{
				"message":    "Payment initiated",
				"gatewayUrl": t.getGatewayURL(session),
			},
		},
	}, nil
}

func (t *PaymentTask) successHandler(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionSuccess) {
		return nil, fmt.Errorf("payment: action %q not permitted", PaymentActionSuccess)
	}

	if err := t.api.Transition(PaymentActionSuccess); err != nil {
		return nil, err
	}

	return &ExecutionResponse{
		Message: "Payment completed successfully",
		ApiResponse: &ApiResponse{
			Success: true,
		},
	}, nil
}

func (t *PaymentTask) failedHandler(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionFailed) {
		return nil, fmt.Errorf("payment: action %q not permitted", PaymentActionFailed)
	}

	record, err := t.createTransactionRecord(ctx)
	if err != nil {
		return nil, err
	}

	newSession := PaymentSession{
		TransactionID: record.ReferenceNumber,
		GeneratedAt:   time.Now(),
	}
	if err := t.api.WriteToLocalStore(paymentStoreSession, &newSession); err != nil {
		return nil, fmt.Errorf("payment: failed to write session: %w", err)
	}

	if err := t.api.Transition(PaymentActionFailed); err != nil {
		return nil, err
	}

	return &ExecutionResponse{
		Message: "Payment failed, new session generated",
		ApiResponse: &ApiResponse{
			Success: true,
		},
	}, nil
}

// ── Helper Methods ────────────────────────────────────────────────────────────

// newSession creates a fresh PaymentSession with a new UUID and the current timestamp.
func (t *PaymentTask) newSession() PaymentSession {
	return PaymentSession{
		TransactionID: uuid.NewString(),
		GeneratedAt:   time.Now(),
	}
}

func (t *PaymentTask) getGatewayURL(session *PaymentSession) string {
	if t.gateway != nil {
		return t.gateway.GenerateRedirectURL(session.TransactionID)
	}

	// Fallback (should not happen if wired correctly)
	return fmt.Sprintf("https://www.lpopp.lk/pay?ref=%s", session.TransactionID)
}

func (t *PaymentTask) readSession(ctx context.Context) (*PaymentSession, error) {
	raw, err := t.api.ReadFromLocalStore(paymentStoreSession)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("no active payment session")
	}

	if s, ok := raw.(*PaymentSession); ok {
		return s, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stored session: %w", err)
	}
	var s PaymentSession
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stored session: %w", err)
	}
	return &s, nil
}

// ── Inquiry Methods ───────────────────────────────────────────────────────────

func (t *PaymentTask) createTransactionRecord(ctx context.Context) (*payment_types.PaymentTransactionDB, error) {
	referenceNumber := uuid.New().String()
	record := payment_types.PaymentTransactionDB{
		ID:              uuid.New(),
		TaskID:          t.api.GetTaskID(),
		ExecutionID:     uuid.New().String(), // Unique ID per payment attempt
		ReferenceNumber: referenceNumber,
		ProviderID:      t.gateway.ID(),
		Status:          "PENDING",
		Amount:          t.config.Amount,
	}

	if err := t.repo.CreateTransaction(ctx, &record); err != nil {
		return nil, fmt.Errorf("payment: failed to persist transaction record: %w", err)
	}
	return &record, nil
}

// GetTransactionByExecutionID retrieves the payment record for a specific FSM execution.
func (t *PaymentTask) GetTransactionByExecutionID(ctx context.Context, execID string) (*payment_types.PaymentTransactionDB, error) {
	return t.repo.GetTransactionByExecutionID(ctx, execID)
}

// GetTransactionByReference retrieves the payment record using the external reference.
func (t *PaymentTask) GetTransactionByReference(ctx context.Context, ref string) (*payment_types.PaymentTransactionDB, error) {
	return t.repo.GetTransactionByReference(ctx, ref, false)
}

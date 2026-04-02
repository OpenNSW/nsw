package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OpenNSW/nsw/internal/payments"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	paymentStoreSession      = "payment:session"
	paymentStoreTransactions = "payment:transactions"
)

// ── Config & Models ───────────────────────────────────────────────────────────

type BreakdownCategory string

const (
	CategoryAddition  BreakdownCategory = "ADDITION"
	CategoryDeduction BreakdownCategory = "DEDUCTION"
)

type BreakdownType string

const (
	TypeFixed      BreakdownType = "FIXED"
	TypePercentage BreakdownType = "PERCENTAGE"
)

type ApplyOn string

// BreakdownItem represents a single line item in the task configuration.
type BreakdownItem struct {
	Description string            `json:"description"`
	Category    BreakdownCategory `json:"category"`
	Type        BreakdownType     `json:"type"`
	Quantity    string            `json:"quantity,omitempty"`  // Placeholder or fixed value
	UnitPrice   string            `json:"unitPrice,omitempty"` // Placeholder or fixed value
	Value       string            `json:"value,omitempty"`     // Percentage value (placeholder or fixed)
}

// ResolvedBreakdownItem is the calculated result sent to the UI.
type ResolvedBreakdownItem struct {
	Description string            `json:"description"`
	Category    BreakdownCategory `json:"category"`
	Type        BreakdownType     `json:"type"`
	Quantity    decimal.Decimal   `json:"quantity"`
	UnitPrice   decimal.Decimal   `json:"unitPrice"`
	Amount      decimal.Decimal   `json:"amount"`
}

// PaymentConfig holds the task-level configuration supplied at workflow definition time.
type PaymentConfig struct {
	Currency    string          `json:"currency"` // Currency of the payment (e.g. "LKR")
	TTL         int             `json:"ttl"`      // Time-to-live for a payment session in seconds
	OrgID       string          `json:"orgId"`    // Organization ID
	ServiceType string          `json:"serviceType,omitempty"`
	Breakdown   []BreakdownItem `json:"breakdown"`
}

// PaymentSession is the current active payment session persisted in local store.
type PaymentSession struct {
	TransactionID    string     `json:"transactionId"`
	ReferenceNumber  string     `json:"referenceNumber"`
	CheckoutURL      string     `json:"checkoutUrl,omitempty"`
	OrgName          string     `json:"orgName,omitempty"`
	SelectedMethodID string     `json:"selectedMethodId,omitempty"`
	GeneratedAt      time.Time  `json:"generatedAt"`
	InitiatedAt      *time.Time `json:"initiatedAt,omitempty"` // set when INITIATE_PAYMENT is received
}

// PaymentTransaction is an append-only history entry for completed (failed/timed-out)
// payment attempts. Callers can introduce new fields without handler changes.
type PaymentTransaction struct {
	TransactionID   string    `json:"transactionId"`
	ReferenceNumber string    `json:"referenceNumber"`
	InitiatedAt     time.Time `json:"initiatedAt"`
	ResolvedAt      time.Time `json:"resolvedAt"`
	Status          string    `json:"status"` // "FAILED" or "TIMEOUT"
	Round           int       `json:"round"`
}

// PaymentRenderContent is the payload returned inside GetRenderInfoResponse.Content
// when the plugin is in IDLE or IN_PROGRESS.
type PaymentRenderContent struct {
	GatewayURL       string                  `json:"gatewayUrl,omitempty"`
	TotalAmount      decimal.Decimal         `json:"totalAmount"`
	Currency         string                  `json:"currency"`
	ReferenceNumber  string                  `json:"referenceNumber"`
	Breakdown        []ResolvedBreakdownItem `json:"breakdown"`
	OrgID            string                  `json:"orgId,omitempty"`
	Service          any                     `json:"service,omitempty"`
	SelectedMethodID string                  `json:"selectedMethodId,omitempty"`
}

// ── FSM ───────────────────────────────────────────────────────────────────────

// NewPaymentFSM returns the state graph for the payment plugin.
// It allows INITIATE_PAYMENT from both IDLE and IN_PROGRESS to support method switching.
//
// State graph:
//
//	""              ──START────────────────► IDLE          [no task state change]
//	IDLE            ──INITIATE_PAYMENT─────► IN_PROGRESS   [IN_PROGRESS]
//	IN_PROGRESS     ──INITIATE_PAYMENT─────► IN_PROGRESS   [IN_PROGRESS]
//	IN_PROGRESS     ──PAYMENT_SUCCESS──────► COMPLETED     [COMPLETED]
//	IN_PROGRESS     ──PAYMENT_FAILED───────► IDLE          [IN_PROGRESS]
//	IN_PROGRESS     ──PAYMENT_TIMEOUT──────► IDLE          [IN_PROGRESS]
func NewPaymentFSM() *PluginFSM {
	return NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:                               {string(paymentIdle), ""},
		{string(paymentIdle), PaymentActionInitiate}:       {string(paymentInProgress), InProgress},
		{string(paymentInProgress), PaymentActionInitiate}: {string(paymentInProgress), InProgress}, // Support Switching
		{string(paymentInProgress), PaymentActionSuccess}:  {string(paymentCompleted), Completed},
		{string(paymentInProgress), PaymentActionFailed}:   {string(paymentIdle), Initialized},
		{string(paymentInProgress), paymentFSMTimeout}:     {string(paymentIdle), Initialized},
	})
}

// ── Plugin ────────────────────────────────────────────────────────────────────

// PaymentTask implements Plugin for the PAYMENT task type.
type PaymentTask struct {
	api            API
	config         PaymentConfig
	paymentService payments.PaymentService
}

// NewPaymentTask creates a PaymentTask from the raw JSON configuration.
func NewPaymentTask(raw json.RawMessage, paymentService payments.PaymentService) (*PaymentTask, error) {
	var cfg PaymentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("payment: invalid config: %w", err)
	}
	return &PaymentTask{
		config:         cfg,
		paymentService: paymentService,
	}, nil
}

func (t *PaymentTask) Init(api API) {
	t.api = api
}

// ── Start ─────────────────────────────────────────────────────────────────────

func (t *PaymentTask) Start(_ context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(FSMActionStart) {
		return &ExecutionResponse{Message: "Payment task already started"}, nil
	}

	session := t.newSession()
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

	resolvedBreakdown, totalAmount, err := t.calculateBreakdown(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to calculate breakdown: %w", err)
	}

	// Terminal state — nothing actionable to render.
	if pluginState == string(paymentCompleted) {
		return &ApiResponse{
			Success: true,
			Data: GetRenderInfoResponse{
				Type:        TaskTypePayment,
				PluginState: pluginState,
				State:       t.api.GetTaskState(),
				Content: PaymentRenderContent{
					TotalAmount: totalAmount,
					Currency:    t.config.Currency,
					Breakdown:   resolvedBreakdown,
					OrgID:       t.config.OrgID,
					Service:     t.config.ServiceType,
				},
			},
		}, nil
	}

	session, err := t.readSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to read session: %w", err)
	}

	// Lazy timeout check: if we are IN_PROGRESS and the payment window has elapsed,
	// transition back to IDLE and record the timeout.
	if pluginState == string(paymentInProgress) && session.InitiatedAt != nil {
		deadline := session.InitiatedAt.Add(t.ttlDuration() + PaymentThreshold)
		if time.Now().After(deadline) {
			if err := t.recordTransaction(ctx, session.TransactionID, session.ReferenceNumber, *session.InitiatedAt, "TIMEOUT"); err != nil {
				return nil, fmt.Errorf("payment: failed to record timeout transaction: %w", err)
			}
			if err := t.api.Transition(paymentFSMTimeout); err != nil {
				return nil, fmt.Errorf("payment: failed timeout transition: %w", err)
			}
			// Refresh plugin state after transition.
			pluginState = t.api.GetPluginState()
		}
	}

	// Rotate session if TTL has elapsed (applies to both IDLE and refreshed-from-timeout).
	if time.Now().After(session.GeneratedAt.Add(t.ttlDuration())) {
		newSess := t.newSession()
		session = &newSess
		if err := t.api.WriteToLocalStore(paymentStoreSession, session); err != nil {
			return nil, fmt.Errorf("payment: failed to persist rotated session: %w", err)
		}
	}

	gatewayURL := session.CheckoutURL
	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypePayment,
			PluginState: pluginState,
			State:       t.api.GetTaskState(),
			Content: PaymentRenderContent{
				GatewayURL:       gatewayURL,
				TotalAmount:      totalAmount,
				Currency:         t.config.Currency,
				ReferenceNumber:  session.ReferenceNumber,
				Breakdown:        resolvedBreakdown,
				OrgID:            t.config.OrgID,
				Service:          t.config.ServiceType,
				SelectedMethodID: session.SelectedMethodID,
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

// initiateHandler processes INITIATE_PAYMENT: validates the session is still within
// TTL, stamps InitiatedAt, and transitions to IN_PROGRESS.
func (t *PaymentTask) initiateHandler(ctx context.Context, content any) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionInitiate) {
		return nil, fmt.Errorf("payment: action %q not permitted in state %q",
			PaymentActionInitiate, t.api.GetPluginState())
	}

	session, err := t.readSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to read session: %w", err)
	}

	// Reject if the session has expired — frontend should call GetRenderInfo for a fresh URL.
	if time.Now().After(session.GeneratedAt.Add(t.ttlDuration())) {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "SESSION_EXPIRED",
					Message: "Payment session has expired. Please refresh to get a new payment URL.",
				},
			},
		}, fmt.Errorf("payment: session expired, cannot initiate payment")
	}

	// Extract methodId from content
	methodID := "lankapay" // Default
	if contentMap, ok := content.(map[string]any); ok {
		if id, ok := contentMap["methodId"].(string); ok {
			methodID = id
		}
	}

	_, totalAmount, err := t.calculateBreakdown(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to calculate total amount: %w", err)
	}

	// Create real checkout session via PaymentService
	resp, err := t.paymentService.CreateCheckoutSession(ctx, payments.CreateCheckoutRequest{
		Amount:          totalAmount,
		Currency:        t.config.Currency,
		ReferenceNumber: session.ReferenceNumber,
		Metadata: map[string]string{
			"task_id":      t.api.GetTaskID(),
			"method_id":    methodID,
			"org_id":       t.config.OrgID,
			"service_type": t.config.ServiceType,
		},
		ExpiresAt: time.Now().Add(t.ttlDuration()),
	})

	if err != nil {
		return nil, fmt.Errorf("payment: failed to create checkout session: %w", err)
	}

	// Parse initiatedAt from content if provided, otherwise use current time.
	now := time.Now()
	if contentMap, ok := content.(map[string]any); ok {
		if tsStr, ok := contentMap["initiatedAt"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
				now = parsed
			}
		}
	}

	session.InitiatedAt = &now
	session.CheckoutURL = resp.CheckoutURL
	session.SelectedMethodID = methodID
	if err := t.api.WriteToLocalStore(paymentStoreSession, session); err != nil {
		return nil, fmt.Errorf("payment: failed to persist initiated session: %w", err)
	}

	// Only transition if we are not already in IN_PROGRESS (e.g. if switching methods)
	if t.api.GetPluginState() != string(paymentInProgress) {
		if err := t.api.Transition(PaymentActionInitiate); err != nil {
			return nil, err
		}
	}

	return &ExecutionResponse{
		Message: "Payment initiated",
		ApiResponse: &ApiResponse{
			Success: true,
			Data: map[string]any{
				"message":     "Payment initiated",
				"checkoutUrl": resp.CheckoutURL,
				"methodId":    methodID,
			},
		},
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (t *PaymentTask) calculateBreakdown(ctx context.Context) ([]ResolvedBreakdownItem, decimal.Decimal, error) {
	var resolved []ResolvedBreakdownItem
	subtotal := decimal.Zero

	var finalTotal decimal.Decimal
	// Phase 1: Fixed Items
	for _, item := range t.config.Breakdown {
		if item.Type != TypeFixed {
			continue
		}

		qty := t.resolveValue(item.Quantity, decimal.NewFromInt(1))
		price := t.resolveValue(item.UnitPrice, decimal.Zero)
		amount := qty.Mul(price)

		if item.Category == CategoryAddition {
			subtotal = subtotal.Add(amount)
		} else {
			subtotal = subtotal.Sub(amount)
		}

		resolved = append(resolved, ResolvedBreakdownItem{
			Description: t.resolveString(item.Description),
			Category:    item.Category,
			Type:        item.Type,
			Quantity:    qty,
			UnitPrice:   price,
			Amount:      amount.Round(2),
		})
	}

	finalTotal = subtotal

	// Phase 2: Percentage Items
	for _, item := range t.config.Breakdown {
		if item.Type != TypePercentage {
			continue
		}

		percentage := t.resolveValue(item.Value, decimal.Zero)
		amount := finalTotal.Mul(percentage).Div(decimal.NewFromInt(100))

		if item.Category == CategoryAddition {
			finalTotal = finalTotal.Add(amount)
		} else {
			finalTotal = finalTotal.Sub(amount)
		}

		resolved = append(resolved, ResolvedBreakdownItem{
			Description: t.resolveString(item.Description),
			Category:    item.Category,
			Type:        item.Type,
			Amount:      amount.Round(2),
		})
	}

	return resolved, finalTotal.Round(2), nil
}

func (t *PaymentTask) resolveValue(val string, fallback decimal.Decimal) decimal.Decimal {
	if val == "" {
		return fallback
	}

	// If placeholder {path:default}
	if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
		inner := val[1 : len(val)-1]
		parts := strings.Split(inner, ":")
		path := parts[0]

		if len(parts) > 1 {
			if d, err := decimal.NewFromString(parts[1]); err == nil {
				fallback = d
			}
		}

		resolved := t.lookupGlobal(path)
		if resolved == nil {
			return fallback
		}

		switch v := resolved.(type) {
		case float64:
			return decimal.NewFromFloat(v)
		case string:
			if d, err := decimal.NewFromString(v); err == nil {
				return d
			}
		case int:
			return decimal.NewFromInt(int64(v))
		case int64:
			return decimal.NewFromInt(v)
		}
		return fallback
	}

	// Literal value
	if d, err := decimal.NewFromString(val); err == nil {
		return d
	}
	return fallback
}

func (t *PaymentTask) resolveString(val string) string {
	// Simple regex-free placeholder replacement
	for {
		start := strings.Index(val, "{")
		end := strings.Index(val, "}")
		if start == -1 || end == -1 || end < start {
			break
		}

		placeholder := val[start : end+1]
		inner := val[start+1 : end]
		parts := strings.Split(inner, ":")
		path := parts[0]

		resolved := t.lookupGlobal(path)
		replacement := ""
		if resolved != nil {
			replacement = fmt.Sprintf("%v", resolved)
		} else if len(parts) > 1 {
			replacement = parts[1]
		}

		val = strings.Replace(val, placeholder, replacement, 1)
	}
	return val
}

func (t *PaymentTask) lookupGlobal(path string) any {
	keys := strings.Split(path, ".")
	val, ok := t.api.ReadFromGlobalStore(keys[0])
	if !ok {
		return nil
	}

	current := val
	for i := 1; i < len(keys); i++ {
		if m, ok := current.(map[string]any); ok {
			current, ok = m[keys[i]]
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}

// successHandler processes PAYMENT_SUCCESS: transitions to COMPLETED.
func (t *PaymentTask) successHandler(_ context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionSuccess) {
		return nil, fmt.Errorf("payment: action %q not permitted in state %q",
			PaymentActionSuccess, t.api.GetPluginState())
	}

	if err := t.api.Transition(PaymentActionSuccess); err != nil {
		return nil, err
	}

	return &ExecutionResponse{
		Message: "Payment completed successfully",
		ApiResponse: &ApiResponse{
			Success: true,
			Data:    map[string]any{"message": "Payment completed successfully"},
		},
	}, nil
}

// failedHandler processes PAYMENT_FAILED: records the failed transaction,
// generates a new session, and transitions back to IDLE.
func (t *PaymentTask) failedHandler(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(PaymentActionFailed) {
		return nil, fmt.Errorf("payment: action %q not permitted in state %q",
			PaymentActionFailed, t.api.GetPluginState())
	}

	session, err := t.readSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to read session: %w", err)
	}

	// Record the failed transaction in history.
	initiatedAt := time.Now()
	if session.InitiatedAt != nil {
		initiatedAt = *session.InitiatedAt
	}
	if err := t.recordTransaction(ctx, session.TransactionID, session.ReferenceNumber, initiatedAt, "FAILED"); err != nil {
		return nil, fmt.Errorf("payment: failed to record failed transaction: %w", err)
	}

	// Generate a fresh session for the next attempt.
	newSess := t.newSession()
	if err := t.api.WriteToLocalStore(paymentStoreSession, &newSess); err != nil {
		return nil, fmt.Errorf("payment: failed to persist new session after failure: %w", err)
	}

	if err := t.api.Transition(PaymentActionFailed); err != nil {
		return nil, err
	}

	return &ExecutionResponse{
		Message: "Payment failed, new session generated",
		ApiResponse: &ApiResponse{
			Success: true,
			Data:    map[string]any{"message": "Payment failed. A new payment session is available."},
		},
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newSession creates a fresh PaymentSession with a new UUID and the current timestamp.
func (t *PaymentTask) newSession() PaymentSession {
	return PaymentSession{
		TransactionID:   uuid.NewString(),
		ReferenceNumber: fmt.Sprintf("NSW-PAY-%s", uuid.NewString()[:8]),
		GeneratedAt:     time.Now(),
	}
}

// ttlDuration returns the configured TTL as a time.Duration.
func (t *PaymentTask) ttlDuration() time.Duration {
	return time.Duration(t.config.TTL) * time.Second
}

// readSession reads and deserialises the current PaymentSession from local store.
// It handles the JSON round-trip that occurs on a cache miss (map[string]any → PaymentSession).
func (t *PaymentTask) readSession(_ context.Context) (*PaymentSession, error) {
	raw, err := t.api.ReadFromLocalStore(paymentStoreSession)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("no active payment session")
	}

	// Fast path: already the correct type (in-memory cache hit).
	if s, ok := raw.(PaymentSession); ok {
		return &s, nil
	}

	// Slow path: JSON round-trip after cache miss / persistence reload.
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to marshal stored session: %w", err)
	}
	var s PaymentSession
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("payment: failed to unmarshal stored session: %w", err)
	}
	return &s, nil
}

// readTransactionHistory reads and deserialises the payment transaction history from local store.
func (t *PaymentTask) readTransactionHistory(_ context.Context) ([]PaymentTransaction, error) {
	raw, err := t.api.ReadFromLocalStore(paymentStoreTransactions)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil // no history yet
	}

	// Fast path.
	if h, ok := raw.([]PaymentTransaction); ok {
		return h, nil
	}

	// Slow path: JSON round-trip.
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("payment: failed to marshal stored transactions: %w", err)
	}
	var h []PaymentTransaction
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, fmt.Errorf("payment: failed to unmarshal stored transactions: %w", err)
	}
	return h, nil
}

// recordTransaction appends a PaymentTransaction to the local store history.
func (t *PaymentTask) recordTransaction(ctx context.Context, transactionID, referenceNumber string, initiatedAt time.Time, status string) error {
	history, err := t.readTransactionHistory(ctx)
	if err != nil {
		return fmt.Errorf("payment: failed to read transaction history: %w", err)
	}
	entry := PaymentTransaction{
		TransactionID:   transactionID,
		ReferenceNumber: referenceNumber,
		InitiatedAt:     initiatedAt,
		ResolvedAt:      time.Now(),
		Status:          status,
		Round:           len(history) + 1,
	}
	history = append(history, entry)
	if err := t.api.WriteToLocalStore(paymentStoreTransactions, history); err != nil {
		return fmt.Errorf("payment: failed to persist transaction history: %w", err)
	}
	return nil
}

package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

type MockPaymentRepo struct {
	mock.Mock
}

func (m *MockPaymentRepo) CreateTransaction(ctx context.Context, trx *payment_types.PaymentTransactionDB) error {
	args := m.Called(ctx, trx)
	return args.Error(0)
}

func (m *MockPaymentRepo) GetTransactionByReference(ctx context.Context, ref string, forUpdate bool) (*payment_types.PaymentTransactionDB, error) {
	args := m.Called(ctx, ref, forUpdate)
	return args.Get(0).(*payment_types.PaymentTransactionDB), args.Error(1)
}

func (m *MockPaymentRepo) GetTransactionByExecutionID(ctx context.Context, execID string) (*payment_types.PaymentTransactionDB, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(*payment_types.PaymentTransactionDB), args.Error(1)
}

func (m *MockPaymentRepo) UpdateTransactionStatus(ctx context.Context, ref string, status string) error {
	args := m.Called(ctx, ref, status)
	return args.Error(0)
}

type MockTaskManager struct {
	mock.Mock
}

func (m *MockTaskManager) InitTask(ctx context.Context, req manager.InitTaskRequest) (*manager.InitTaskResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*manager.InitTaskResponse), args.Error(1)
}

func (m *MockTaskManager) ExecuteTask(ctx context.Context, req manager.ExecuteTaskRequest) (*plugin.ExecutionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.ExecutionResponse), args.Error(1)
}

func (m *MockTaskManager) GetTaskRenderInfo(ctx context.Context, taskID string) (*plugin.ApiResponse, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.ApiResponse), args.Error(1)
}

func (m *MockTaskManager) RegisterUpstreamCallback(cb manager.WorkflowUpdateHandler) {
	m.Called(cb)
}

// Rewriting test to use a fully mocked gateway for better control
type MockGatewayInterface struct {
	mock.Mock
}

func (m *MockGatewayInterface) ID() string { return "mock-gw" }
func (m *MockGatewayInterface) GenerateRedirectURL(ctx context.Context, trx *payment_types.PaymentTransactionDB, returnUrl string) (string, error) {
	return "", nil
}
func (m *MockGatewayInterface) ExtractReference(r *http.Request) (string, error) {
	args := m.Called(r)
	return args.String(0), args.Error(1)
}
func (m *MockGatewayInterface) GetPaymentInfo(ctx context.Context, ref string) (gateway.CallbackResult, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).(gateway.CallbackResult), args.Error(1)
}
func (m *MockGatewayInterface) FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error) {
	return nil, nil
}

func TestProcessCallback_SecureVerificationFlow(t *testing.T) {
	db, sqlMock, _ := sqlmock.New()
	gdb, _ := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})

	gw := new(MockGatewayInterface)
	registry := gateway.NewRegistry()
	registry.Register(gw)

	repo := new(MockPaymentRepo)
	tm := new(MockTaskManager)

	svc := NewService(registry, repo, tm, gdb)

	ctx := context.Background()
	req := httptest.NewRequest("POST", "/callback", nil)
	refNo := "REF123"
	taskID := uuid.New()

	// ExtractReference
	gw.On("ExtractReference", req).Return(refNo, nil)

	// GetPaymentInfo
	gw.On("GetPaymentInfo", ctx, refNo).Return(gateway.CallbackResult{
		ReferenceNumber: refNo,
		ProviderID:      "mock-gw",
		Status:          "SUCCESS",
	}, nil)

	// DB Transaction
	sqlMock.ExpectBegin()
	trxRecord := &payment_types.PaymentTransactionDB{
		ID:              uuid.New(),
		TaskID:          taskID,
		ReferenceNumber: refNo,
		Status:          "PENDING",
	}
	repo.On("GetTransactionByReference", ctx, refNo, true).Return(trxRecord, nil)
	sqlMock.ExpectExec("UPDATE \"payment_transactions\"").
		WithArgs("COMPLETED", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	// ExecuteTask
	tm.On("ExecuteTask", ctx, manager.ExecuteTaskRequest{
		TaskID:  taskID,
		Payload: &plugin.ExecutionRequest{Action: plugin.PaymentActionSuccess},
	}).Return(&plugin.ExecutionResponse{}, nil)

	err := svc.ProcessCallback(ctx, "mock-gw", req)
	assert.NoError(t, err)

	gw.AssertExpectations(t)
	repo.AssertExpectations(t)
	tm.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

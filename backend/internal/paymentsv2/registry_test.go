package paymentsv2

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/OpenNSW/nsw/backend/internal/paymentsv2/gateways"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGateway is a mock implementation of gateways.PaymentGateway
type MockGateway struct {
	mock.Mock
}

func (m *MockGateway) ApplyConfig(config json.RawMessage) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockGateway) GetFlowType() gateways.InteractionType {
	args := m.Called()
	return args.Get(0).(gateways.InteractionType)
}

func (m *MockGateway) CreateSession(ctx context.Context, req gateways.SessionRequest) (*gateways.SessionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gateways.SessionResponse), args.Error(1)
}

func (m *MockGateway) ExtractReferenceNumber(ctx context.Context, reqData json.RawMessage) (string, error) {
	args := m.Called(ctx, reqData)
	return args.String(0), args.Error(1)
}

func (m *MockGateway) HandleValidateReference(ctx context.Context, tx *gateways.ValidationTransaction, isPayable bool, reqData json.RawMessage) (*gateways.ValidationResponse, error) {
	args := m.Called(ctx, tx, isPayable, reqData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gateways.ValidationResponse), args.Error(1)
}

func (m *MockGateway) ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*gateways.WebhookPayload, error) {
	args := m.Called(ctx, body, headers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gateways.WebhookPayload), args.Error(1)
}

func TestNewRegistry(t *testing.T) {
	// Setup temporary config file
	configContent := `{
		"version": "1.0",
		"methods": [
			{
				"id": "lankapay",
				"is_active": true,
				"render_info": {
					"display_name": "LankaPay",
					"display_order": 1
				},
				"config": {"apiKey": "secret"}
			}
		]
	}`
	tmpFile, err := os.CreateTemp("", "payment_methods_*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(configContent)
	assert.NoError(t, err)
	tmpFile.Close()

	mockG := new(MockGateway)
	mockG.On("ApplyConfig", json.RawMessage(`{"apiKey": "secret"}`)).Return(nil)

	gateways := map[string]gateways.PaymentGateway{
		"lankapay": mockG,
	}

	registry, err := NewRegistry(tmpFile.Name(), gateways)
	assert.NoError(t, err)
	assert.NotNil(t, registry)

	mockG.AssertExpectations(t)
}

func TestListInfo(t *testing.T) {
	configContent := `{
		"version": "1.0",
		"methods": [
			{
				"id": "method2",
				"is_active": true,
				"render_info": {
					"display_name": "Method 2",
					"display_order": 2
				},
				"config": {"secret": "keep-away"}
			},
			{
				"id": "method1",
				"is_active": true,
				"render_info": {
					"display_name": "Method 1",
					"display_order": 1
				}
			},
			{
				"id": "inactive",
				"is_active": false,
				"render_info": {
					"display_name": "Inactive",
					"display_order": 3
				}
			}
		]
	}`
	tmpFile, err := os.CreateTemp("", "payment_methods_*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	os.WriteFile(tmpFile.Name(), []byte(configContent), 0644)

	registry, _ := NewRegistry(tmpFile.Name(), map[string]gateways.PaymentGateway{})
	infos := registry.ListInfo()

	// 1. Should only contain active methods
	assert.Equal(t, 2, len(infos))

	// 2. Should be sorted by DisplayOrder (Method 1 should come before Method 2)
	assert.Equal(t, "method1", infos[0].ID)
	assert.Equal(t, "method2", infos[1].ID)

	// 3. Should be sanitized (Config should be empty)
	assert.Nil(t, infos[1].Config)
}

func TestGet(t *testing.T) {
	configContent := `{
		"version": "1.0",
		"methods": [
			{
				"id": "gw1",
				"is_active": true
			}
		]
	}`
	tmpFile, _ := os.CreateTemp("", "payment_methods_*.json")
	defer os.Remove(tmpFile.Name())
	os.WriteFile(tmpFile.Name(), []byte(configContent), 0644)

	mockG := new(MockGateway)
	registry, _ := NewRegistry(tmpFile.Name(), map[string]gateways.PaymentGateway{"gw1": mockG})

	gateway, err := registry.Get("gw1")
	assert.NoError(t, err)
	assert.Equal(t, mockG, gateway)

	_, err = registry.Get("non-existent")
	assert.Error(t, err)
}

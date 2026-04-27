package payments

import (
	"context"
	"errors"
	"testing"
)

// mockRepository implements PaymentRepository for testing.
type mockRepository struct {
	PaymentRepository
	txByRef map[string]*PaymentTransaction
	getErr  error
}

func (m *mockRepository) GetByReferenceNumber(ctx context.Context, ref string) (*PaymentTransaction, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.txByRef[ref], nil
}

// mockRegistry implements PaymentRegistry for testing.
type mockRegistry struct {
	PaymentRegistry
	providers   map[string]PaymentProvider
	infoList    []PaymentProviderInfo
	defaultProv PaymentProvider
}

func (m *mockRegistry) Get(id string) (PaymentProvider, error) {
	p, ok := m.providers[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}

func (m *mockRegistry) ListInfo() []PaymentProviderInfo {
	return m.infoList
}

// mockProvider implements PaymentProvider for testing.
type mockProvider struct {
	PaymentProvider
	info        PaymentProviderInfo
	validateRes *ValidateReferenceResponse
	validateErr error
}

func (m *mockProvider) RenderInfo() PaymentRenderInfo {
	return m.info.RenderInfo
}

func (m *mockProvider) HandleValidateReference(ctx context.Context, tx *PaymentTransaction) (*ValidateReferenceResponse, error) {
	return m.validateRes, m.validateErr
}

func TestService_ListAvailableMethods(t *testing.T) {
	expectedInfo := []PaymentProviderInfo{
		{ID: "p1", IsActive: true, RenderInfo: PaymentRenderInfo{DisplayName: "Provider 1"}},
	}
	registry := &mockRegistry{infoList: expectedInfo}
	service := NewPaymentService(nil, registry)

	res, err := service.ListAvailableMethods(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 || res[0].ID != "p1" {
		t.Errorf("expected p1, got %v", res)
	}
}

func TestService_ValidateReference(t *testing.T) {
	providerID := "lankapay"
	ref := "REF-123"

	repo := &mockRepository{txByRef: make(map[string]*PaymentTransaction)}
	registry := &mockRegistry{providers: make(map[string]PaymentProvider)}
	service := NewPaymentService(repo, registry)

	t.Run("success", func(t *testing.T) {
		tx := &PaymentTransaction{ReferenceNumber: ref, ProviderID: providerID}
		repo.txByRef[ref] = tx

		prov := &mockProvider{validateRes: &ValidateReferenceResponse{IsPayable: true}}
		registry.providers[providerID] = prov

		res, err := service.ValidateReference(context.Background(), providerID, ValidateReferenceRequest{PaymentReference: ref})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsPayable {
			t.Error("expected IsPayable to be true")
		}
	})

	t.Run("not found in db", func(t *testing.T) {
		delete(repo.txByRef, ref)
		res, err := service.ValidateReference(context.Background(), providerID, ValidateReferenceRequest{PaymentReference: ref})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsPayable {
			t.Error("expected IsPayable to be false for non-existent ref")
		}
	})

	t.Run("provider mismatch", func(t *testing.T) {
		repo.txByRef[ref] = &PaymentTransaction{ReferenceNumber: ref, ProviderID: "different-provider"}
		registry.providers[providerID] = &mockProvider{}

		res, err := service.ValidateReference(context.Background(), providerID, ValidateReferenceRequest{PaymentReference: ref})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsPayable {
			t.Error("expected IsPayable to be false for provider mismatch")
		}
		if res.Remarks != "Reference mismatch" {
			t.Errorf("expected 'Reference mismatch', got %s", res.Remarks)
		}
	})

	t.Run("registry error", func(t *testing.T) {
		delete(registry.providers, providerID)
		_, err := service.ValidateReference(context.Background(), providerID, ValidateReferenceRequest{PaymentReference: ref})
		if err == nil {
			t.Error("expected error for missing provider in registry")
		}
	})
}

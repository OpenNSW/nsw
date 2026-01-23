package form

import (
	"context"
	"errors"
	"fmt"

	formmodel "github.com/OpenNSW/nsw/internal/form/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrFormNotFound is returned when a form is not found
var ErrFormNotFound = errors.New("form not found")

// FormService provides methods to retrieve form definitions
// FormService is a pure domain service that only works with forms.
// It has no knowledge of tasks, task types, or task configurations.
// Task-related operations should be handled by TaskManager, which will call FormService.GetFormByID.
type FormService interface {
	// GetFormByID retrieves a form by its UUID
	// Returns the JSON Schema and UI Schema that portals can directly use with JSON Forms
	GetFormByID(ctx context.Context, formID uuid.UUID) (*formmodel.FormResponse, error)
}

type formService struct {
	db *gorm.DB
}

// NewFormService creates a new FormService instance
func NewFormService(db *gorm.DB) FormService {
	return &formService{
		db: db,
	}
}

// GetFormByID retrieves a form by its UUID
func (s *formService) GetFormByID(ctx context.Context, formID uuid.UUID) (*formmodel.FormResponse, error) {
	if formID == uuid.Nil {
		return nil, fmt.Errorf("formID cannot be nil")
	}

	var form formmodel.Form
	if err := s.db.WithContext(ctx).
		Where("id = ? AND active = ?", formID, true).
		First(&form).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("form with ID %s not found: %w", formID, ErrFormNotFound)
		}
		return nil, fmt.Errorf("failed to retrieve form: %w", err)
	}

	return &formmodel.FormResponse{
		ID:       form.ID,
		Name:     form.Name,
		Schema:   form.Schema,
		UISchema: form.UISchema,
		Version:  form.Version,
	}, nil
}

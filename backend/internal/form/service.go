package form

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	formmodel "github.com/OpenNSW/nsw/internal/form/model"
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
	GetFormByID(ctx context.Context, formID string) (*formmodel.FormResponse, error)
}

type formService struct {
	formsPath string
}

// NewFormService creates a new FormService instance that reads forms from the filesystem
func NewFormService(formsPath string) FormService {
	return &formService{
		formsPath: formsPath,
	}
}

// GetFormByID retrieves a form by its UUID from the filesystem
func (s *formService) GetFormByID(ctx context.Context, formID string) (*formmodel.FormResponse, error) {
	if formID == "" {
		return nil, fmt.Errorf("formID cannot be empty")
	}

	formFilePath := filepath.Join(s.formsPath, fmt.Sprintf("%s.json", formID))
	data, err := os.ReadFile(formFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("form with ID %s not found in %s: %w", formID, s.formsPath, ErrFormNotFound)
		}
		return nil, fmt.Errorf("failed to read form file: %w", err)
	}

	var formResponse formmodel.FormResponse
	if err := json.Unmarshal(data, &formResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal form data: %w", err)
	}

	// Ensure ID matches the requested formID if it's set in the file, or set it if not
	if formResponse.ID == "" {
		formResponse.ID = formID
	}

	return &formResponse, nil
}

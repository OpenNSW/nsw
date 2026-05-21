package company

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"gorm.io/gorm"
)

// Service defines operations for company profile management.
type Service interface {
	// GetCompanyByID retrieves a company record by its ID.
	// Returns ErrCompanyNotFound if no record exists.
	GetCompanyByID(ctx context.Context, id string) (*Record, error)

	// GetCompanyByOUHandle retrieves a company record by its IdP organisational unit handle.
	// Returns ErrCompanyNotFound if no record exists.
	GetCompanyByOUHandle(ctx context.Context, ouHandle string) (*Record, error)

	// ListCompanies returns company records ordered by name, optionally filtered by HasCHA and a
	// case-insensitive substring match on Name.
	ListCompanies(ctx context.Context, filter ListFilter) ([]Record, error)

	// UpdateCompany performs an append-only merge of data into the company's Data field.
	// New keys are added and existing keys are replaced only when explicitly provided.
	// Keys absent from data are never removed.
	// Returns ErrCompanyNotFound if the company does not exist.
	UpdateCompany(ctx context.Context, id string, data map[string]any) error

	// Health checks if the service can access the database.
	Health(ctx context.Context) error
}

type service struct {
	db *gorm.DB
}

// NewService creates a new company service instance.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) getByField(ctx context.Context, field, value string) (*Record, error) {
	var record Record
	result := s.db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", field), value).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("company record not found", field, value)
			return nil, ErrCompanyNotFound
		}
		slog.Error("failed to fetch company record", field, value, "error", result.Error)
		return nil, fmt.Errorf("database query failed: %w", result.Error)
	}
	return &record, nil
}

func (s *service) GetCompanyByID(ctx context.Context, id string) (*Record, error) {
	if id == "" {
		return nil, ErrInvalidCompanyID
	}
	return s.getByField(ctx, "id", id)
}

func (s *service) GetCompanyByOUHandle(ctx context.Context, ouHandle string) (*Record, error) {
	return s.getByField(ctx, "ou_handle", ouHandle)
}

func (s *service) ListCompanies(ctx context.Context, filter ListFilter) ([]Record, error) {
	q := s.db.WithContext(ctx).Model(&Record{})
	if filter.HasCHA != nil {
		q = q.Where("has_cha = ?", *filter.HasCHA)
	}
	if filter.Name != nil {
		name := strings.TrimSpace(*filter.Name)
		if name != "" {
			q = q.Where("name ILIKE ?", "%"+name+"%")
		}
	}

	var records []Record
	if err := q.Order("name ASC").Find(&records).Error; err != nil {
		slog.Error("failed to list company records", "error", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	return records, nil
}

func (s *service) UpdateCompany(ctx context.Context, id string, data map[string]any) error {
	if id == "" {
		return ErrInvalidCompanyID
	}

	if len(data) == 0 {
		return nil
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal company data: %w", err)
	}

	// Use PostgreSQL's JSONB concatenation operator (||) for an atomic merge.
	// This avoids race conditions inherent in a read-modify-write cycle.
	result := s.db.WithContext(ctx).Model(&Record{}).
		Where("id = ?", id).
		Update("data", gorm.Expr("data || ?::jsonb", string(jsonBytes)))

	if result.Error != nil {
		slog.Error("failed to update company data", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update company data: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

func (s *service) Health(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		slog.Error("failed to retrieve underlying sql db", "error", err)
		return fmt.Errorf("failed to retrieve database: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		slog.Error("company service health check failed", "error", err)
		return fmt.Errorf("company service health check failed: %w", err)
	}
	return nil
}

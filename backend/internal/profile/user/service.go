package user

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Service defines operations for user profile management.
// It handles database interactions for persisted user records.
type Service interface {
	// GetUser retrieves a user record by ID.
	// Returns a user record if found, nil if not found, or an error on failure.
	GetUser(userID string) (*Record, error)

	// CreateUser creates a new user record.
	// Returns an error if the user already exists or if the operation fails.
	CreateUser(userID, email, phone, ouID string) error

	// UpdateUserNSWData updates the NSWData field for an existing user record.
	// The provided data should be valid JSON bytes.
	// Returns ErrUserNotFound if the user does not exist.
	UpdateUserNSWData(userID string, nswData []byte) error

	// UserExists checks if a user record exists for the given userID.
	UserExists(userID string) (bool, error)

	// Health checks if the service can access the database.
	Health() error
}

// service implements the Service interface using GORM.
type service struct {
	db *gorm.DB
}

// NewService creates a new user service instance.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

// GetUser retrieves a user record from the database.
func (s *service) GetUser(userID string) (*Record, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	var record Record
	result := s.db.Where("user_id = ?", userID).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("user record not found", "user_id", userID)
			return nil, ErrUserNotFound
		}
		slog.Error("failed to fetch user record", "user_id", userID, "error", result.Error)
		return nil, fmt.Errorf("database query failed: %w", result.Error)
	}

	return &record, nil
}

// CreateUser creates a new user record in the database.
// The NSWData field is initialized to an empty JSON object.
func (s *service) CreateUser(userID, email, phone, ouID string) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	record := &Record{
		UserID:      userID,
		Email:       email,
		PhoneNumber: phone,
		OUID:        ouID,
		NSWData:     []byte(`{}`),
	}

	result := s.db.Create(record)
	if result.Error != nil {
		slog.Error("failed to create user record", "user_id", userID, "error", result.Error)
		return fmt.Errorf("database insert failed: %w", result.Error)
	}

	slog.Debug("user record created", "user_id", userID, "email", email)
	return nil
}

// Health checks if the service can access the database.
func (s *service) Health() error {
	var count int64
	result := s.db.Model(&Record{}).Count(&count)
	if result.Error != nil {
		slog.Error("user service health check failed", "error", result.Error)
		return fmt.Errorf("user service health check failed: %w", result.Error)
	}

	slog.Debug("user service health check passed", "accessible", true)
	return nil
}

// UpdateUserNSWData updates the NSWData field for a user record.
func (s *service) UpdateUserNSWData(userID string, nswData []byte) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	result := s.db.Model(&Record{}).Where("user_id = ?", userID).Update("nsw_data", nswData)
	if result.Error != nil {
		slog.Error("failed to update user NSWData", "user_id", userID, "error", result.Error)
		return fmt.Errorf("database update failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		slog.Debug("user record not found for update", "user_id", userID)
		return ErrUserNotFound
	}

	slog.Debug("user NSWData updated", "user_id", userID)
	return nil
}

// UserExists checks if a user record exists for the given userID.
func (s *service) UserExists(userID string) (bool, error) {
	slog.Info("checking if user exists", "user_id", userID)
	if userID == "" {
		return false, ErrInvalidUserID
	}

	var count int64
	result := s.db.Model(&Record{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("user record not found during existence check", "user_id", userID)
			return false, nil
		}
		slog.Error("failed to check if user exists", "user_id", userID, "error", result.Error)
		return false, fmt.Errorf("database query failed: %w", result.Error)
	}

	exists := count > 0
	slog.Debug("user existence check", "user_id", userID, "exists", exists)
	return exists, nil
}

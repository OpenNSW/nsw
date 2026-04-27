package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuthService provides business logic for authentication and user context operations.
// It handles database interactions for user-related data.
type AuthService struct {
	db *gorm.DB
}

// NewAuthService creates a new AuthService instance
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// GetUserContext retrieves the user context from the database for a given user ID.
// Returns gorm.ErrRecordNotFound when the user has no context entry.
//
// This method is responsible for:
// 1. Database lookup of user context
// 2. Handling not found errors gracefully
// 3. Logging errors for debugging
func (as *AuthService) GetUserContext(userID string) (*UserContext, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is empty")
	}

	var userRecord UserRecord
	result := as.db.Where("user_id = ?", userID).First(&userRecord)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("user context not found", "user_id", userID)
			return nil, result.Error
		}
		return nil, fmt.Errorf("failed to fetch user context: %w", result.Error)
	}

	return &UserContext{
		UserID:      userRecord.UserID,
		Email:       userRecord.Email,
		PhoneNumber: userRecord.PhoneNumber,
		OUID:        userRecord.OUID,
		Roles:       []string{},
		NSWData:     userRecord.NSWData,
	}, nil
}

// UpdateUserContext updates the user context for a given user ID.
// The context parameter should be a valid JSON serialized to json.RawMessage.
//
// This method handles:
// 1. Validation of user existence
// 2. Database update of user context
// 3. Error handling and logging
//
// Example usage:
//
//	newContext := json.RawMessage(`{"company": "Acme Inc", "role": "exporter"}`)
//	err := authService.UpdateUserContext("TRADER-001", newContext)
//
// TODO: Enhancements to consider:
// - Audit logging for who/when/what changed
// - Version tracking for user context changes
// - Webhook notifications on context updates
func (as *AuthService) UpdateUserContext(userID string, ctx json.RawMessage) error {
	if userID == "" {
		return fmt.Errorf("user ID is empty")
	}
	if len(ctx) == 0 {
		return fmt.Errorf("user context is empty")
	}

	var jsonData interface{}
	if err := json.Unmarshal(ctx, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON in user context: %w", err)
	}

	result := as.db.Model(&UserRecord{}).
		Where("user_id = ?", userID).
		Update("nsw_data", ctx)
	if result.Error != nil {
		return fmt.Errorf("failed to update user context: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user context not found for user_id: %s", userID)
	}
	return nil
}

type UpsertUserContextPayload struct {
	Email       *string
	PhoneNumber *string
	OUID        *string
	NSWData     json.RawMessage
}

// UpsertUserContext creates or updates the user context.
// If the user doesn't exist, it will be created with the provided context.
// If it exists, the context will be updated.
//
// This is useful for initialization or bulk operations.
//
// TODO: This method might be useful when:
// - Receiving user context updates from external Identity systems
// - Initializing new users during registration
// - Syncing with identity management systems
func (as *AuthService) UpsertUserContext(userID string, payload UpsertUserContextPayload) error {
	if userID == "" {
		return fmt.Errorf("user ID is empty")
	}

	defaultNSWData := json.RawMessage(`{}`)
	if len(payload.NSWData) == 0 {
		payload.NSWData = defaultNSWData
	}

	userRecord := &UserRecord{
		UserID:  userID,
		NSWData: payload.NSWData,
	}

	userContext, err := as.GetUserContext(userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to upsert user context: %w", err)
		}
	} else {
		userRecord.Email = userContext.Email
		userRecord.PhoneNumber = userContext.PhoneNumber
		userRecord.OUID = userContext.OUID
	}

	if payload.Email != nil {
		userRecord.Email = *payload.Email
	}
	if payload.PhoneNumber != nil {
		userRecord.PhoneNumber = *payload.PhoneNumber
	}
	if payload.OUID != nil {
		userRecord.OUID = *payload.OUID
	}

	result := as.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"email", "phone_number", "ou_id", "nsw_data"}),
	}).Create(userRecord)
	if result.Error != nil {
		return fmt.Errorf("failed to upsert user context: %w", result.Error)
	}
	return nil
}

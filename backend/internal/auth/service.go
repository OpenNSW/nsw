package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
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
//
// TODO_JWT_FUTURE: When JWT is implemented:
// - This method will still be called the same way
// - No changes needed here, token verification happens in token_parser.go
// - Consider caching user contexts for performance optimization
func (as *AuthService) GetUserContext(userID string) (*UserContext, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is empty")
	}

	var uc UserContext
	result := as.db.Where("user_id = ?", userID).First(&uc)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("user context not found", "user_id", userID)
			return nil, result.Error
		}
		return nil, fmt.Errorf("failed to fetch user context: %w", result.Error)
	}

	return &uc, nil
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
// TODO_JWT_FUTURE: Consider adding:
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

	result := as.db.Model(&UserContext{}).
		Where("user_id = ?", userID).
		Update("user_context", ctx)
	if result.Error != nil {
		return fmt.Errorf("failed to update user context: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user context not found for user_id: %s", userID)
	}
	return nil
}

// UpsertUserContext creates or updates the user context.
// If the user doesn't exist, it will be created with the provided context.
// If it exists, the context will be updated.
//
// This is useful for initialization or bulk operations.
//
// TODO_JWT_FUTURE: This method might be useful when:
// - Receiving user context updates from external systems
// - Initializing new users during registration
// - Syncing with identity management systems
func (as *AuthService) UpsertUserContext(userID string, ctx json.RawMessage) error {
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

	result := as.db.Save(&UserContext{
		UserID:      userID,
		UserContext: ctx,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to upsert user context: %w", result.Error)
	}
	return nil
}

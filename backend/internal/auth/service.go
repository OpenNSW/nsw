package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// AuthService provides business logic for authentication and trader context operations.
// It handles database interactions for trader-related data.
type AuthService struct {
	db *gorm.DB
}

// NewAuthService creates a new AuthService instance
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db: db,
	}
}

// GetTraderContext retrieves the trader context from the database for a given trader ID.
// Returns nil if the trader is not found (indicating an unauthorized request).
//
// This method is responsible for:
// 1. Database lookup of trader context
// 2. Handling not found errors gracefully
// 3. Logging errors for debugging
//
// TODO_JWT_FUTURE: When JWT is implemented:
// - This method will still be called the same way
// - No changes needed here, token verification happens in token_parser.go
// - Consider caching trader contexts for performance optimization
func (as *AuthService) GetTraderContext(traderID string) (*TraderContext, error) {
	if traderID == "" {
		return nil, fmt.Errorf("trader ID is empty")
	}

	var traderCtx TraderContext
	result := as.db.Where("trader_id = ?", traderID).First(&traderCtx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("trader context not found", "trader_id", traderID)
			return nil, result.Error
		}
		slog.Error("failed to fetch trader context from database",
			"trader_id", traderID,
			"error", result.Error,
		)
		return nil, fmt.Errorf("failed to fetch trader context: %w", result.Error)
	}

	return &traderCtx, nil
}

// UpdateTraderContext updates the trader context for a given trader ID.
// The context parameter should be a valid JSON serialized to json.RawMessage.
//
// This method handles:
// 1. Validation of trader existence
// 2. Database update of trader context
// 3. Error handling and logging
//
// Example usage:
//
//	newContext := json.RawMessage(`{"company": "Acme Inc", "role": "exporter"}`)
//	err := authService.UpdateTraderContext("TRADER-001", newContext)
//
// TODO_JWT_FUTURE: Consider adding:
// - Audit logging for who/when/what changed
// - Version tracking for trader context changes
// - Webhook notifications on context updates
func (as *AuthService) UpdateTraderContext(traderID string, context json.RawMessage) error {
	if traderID == "" {
		return fmt.Errorf("trader ID is empty")
	}

	if len(context) == 0 {
		return fmt.Errorf("trader context is empty")
	}

	// Validate JSON format
	var jsonData interface{}
	if err := json.Unmarshal(context, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON in trader context: %w", err)
	}

	// Update the trader context in the database
	result := as.db.Model(&TraderContext{}).
		Where("trader_id = ?", traderID).
		Update("trader_context", context)

	if result.Error != nil {
		slog.Error("failed to update trader context in database",
			"trader_id", traderID,
			"error", result.Error,
		)
		return fmt.Errorf("failed to update trader context: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		slog.Warn("no trader context found to update",
			"trader_id", traderID,
		)
		return fmt.Errorf("trader context not found for trader_id: %s", traderID)
	}

	slog.Debug("trader context updated successfully",
		"trader_id", traderID,
		"rows_affected", result.RowsAffected,
	)

	return nil
}

// UpsertTraderContext creates or updates the trader context.
// If the trader doesn't exist, it will be created with the provided context.
// If it exists, the context will be updated.
//
// This is useful for initialization or bulk operations.
//
// TODO_JWT_FUTURE: This method might be useful when:
// - Receiving trader context updates from external systems
// - Initializing new traders during registration
// - Syncing with identity management systems
func (as *AuthService) UpsertTraderContext(traderID string, context json.RawMessage) error {
	if traderID == "" {
		return fmt.Errorf("trader ID is empty")
	}

	if len(context) == 0 {
		return fmt.Errorf("trader context is empty")
	}

	// Validate JSON format
	var jsonData interface{}
	if err := json.Unmarshal(context, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON in trader context: %w", err)
	}

	result := as.db.Save(&TraderContext{
		TraderID:      traderID,
		TraderContext: context,
	})

	if result.Error != nil {
		slog.Error("failed to upsert trader context",
			"trader_id", traderID,
			"error", result.Error,
		)
		return fmt.Errorf("failed to upsert trader context: %w", result.Error)
	}

	slog.Debug("trader context upserted successfully",
		"trader_id", traderID,
	)

	return nil
}

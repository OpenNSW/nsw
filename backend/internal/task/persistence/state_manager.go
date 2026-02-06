package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager interface {
	GetState(key string) (any, error)
	SetState(key string, state any) error
}

// LocalStateManager implements the Manager interface for task-specific local state
// It persists state to the TaskStore's local_state column
type LocalStateManager struct {
	taskStore TaskStoreInterface
	taskID    uuid.UUID
	cache     map[string]any // In-memory cache for performance
}

// NewLocalStateManager creates a new LocalStateManager for a specific task
func NewLocalStateManager(taskStore TaskStoreInterface, taskID uuid.UUID) (*LocalStateManager, error) {
	manager := &LocalStateManager{
		taskStore: taskStore,
		taskID:    taskID,
		cache:     make(map[string]any),
	}

	// Load existing state from database
	if err := manager.loadFromDB(); err != nil {
		return nil, fmt.Errorf("failed to load local state: %w", err)
	}

	return manager, nil
}

func NewLocalStateManagerWithCache(taskStore TaskStoreInterface, taskID uuid.UUID, cache json.RawMessage) (*LocalStateManager, error) {
	cacheMap := make(map[string]any)

	if len(cache) > 0 && string(cache) != "null" {
		if err := json.Unmarshal(cache, &cacheMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal local state: %w", err)
		}
	}

	return &LocalStateManager{
		taskStore: taskStore,
		taskID:    taskID,
		cache:     cacheMap,
	}, nil
}

// GetState retrieves a value from local state by key
func (m *LocalStateManager) GetState(key string) (any, error) {
	value, exists := m.cache[key]
	if !exists {
		return nil, nil
	}
	return value, nil
}

// SetState sets a value in local state and persists to database
func (m *LocalStateManager) SetState(key string, value any) error {
	// Update cache
	m.cache[key] = value

	// Persist to database (write-through)
	return m.persistToDB()
}

// loadFromDB loads the local state from the database into cache
func (m *LocalStateManager) loadFromDB() error {
	localStateJSON, err := m.taskStore.GetLocalState(m.taskID)
	if err != nil {
		// If record is not found, it's not an error; we just start with an empty state.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		// For any other error, we should return it.
		return err
	}

	// If empty or nil, start with empty cache
	if len(localStateJSON) == 0 {
		return nil
	}

	// Parse JSON into cache
	if err := json.Unmarshal(localStateJSON, &m.cache); err != nil {
		return fmt.Errorf("failed to unmarshal local state: %w", err)
	}

	return nil
}

// persistToDB writes the current cache to the database
func (m *LocalStateManager) persistToDB() error {
	// Serialize cache to JSON
	localStateJSON, err := json.Marshal(m.cache)
	if err != nil {
		return fmt.Errorf("failed to marshal local state: %w", err)
	}

	// Write to database
	if err := m.taskStore.UpdateLocalState(m.taskID, localStateJSON); err != nil {
		return fmt.Errorf("failed to update local state in database: %w", err)
	}

	return nil
}

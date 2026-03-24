package consignment

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type CHAService struct {
	db *gorm.DB
}

func NewCHAService(db *gorm.DB) *CHAService {
	return &CHAService{db: db}
}

// ListCHAs returns all customs house agents ordered by name.
func (s *CHAService) ListCHAs(ctx context.Context) ([]CHA, error) {
	var chas []CHA
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&chas).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve customs house agents: %w", err)
	}
	return chas, nil
}

// GetCHAByEmail looks up a CHA by its email (used to resolve CHA identity from an auth token).
func (s *CHAService) GetCHAByEmail(ctx context.Context, email string) (*CHA, error) {
	var cha CHA
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&cha).Error; err != nil {
		return nil, fmt.Errorf("failed to find CHA with email %q: %w", email, err)
	}
	return &cha, nil
}

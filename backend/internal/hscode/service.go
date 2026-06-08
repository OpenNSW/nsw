package hscode

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/backend/pkg/pagination"
)

type Service struct {
	db *gorm.DB
}

// NewService creates a new instance of Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// GetAll retrieves all HS codes from the database
func (s *Service) GetAll(ctx context.Context, filter Filter) (*ListResult, error) {
	finalOffset, finalLimit := pagination.ResolvePaginationParams(filter.Offset, filter.Limit)

	// baseQuery builds a fresh *gorm.DB with only the filter conditions applied.
	// A new instance is constructed each time to prevent GORM's shared Clauses map
	// from being contaminated by pagination (LIMIT/OFFSET) set on the list query.
	baseQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&HSCode{})
		if filter.HSCodeStartsWith != nil && *filter.HSCodeStartsWith != "" {
			q = q.Where("hs_code LIKE ?", *filter.HSCodeStartsWith+"%")
		}
		return q
	}

	var hsCodes []HSCode
	if err := baseQuery().Offset(finalOffset).Limit(finalLimit).Order("hs_code ASC").Find(&hsCodes).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve HS codes: %w", err)
	}

	var totalCount int64
	if len(hsCodes) < finalLimit && finalOffset == 0 {
		totalCount = int64(len(hsCodes))
	} else {
		if err := baseQuery().Count(&totalCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count HS codes: %w", err)
		}
	}

	result := pagination.NewPageResult(hsCodes, totalCount, finalOffset, finalLimit)
	return &result, nil
}

// GetByID retrieves an HS code by its ID from the database
func (s *Service) GetByID(ctx context.Context, hsCodeID string) (*HSCode, error) {
	var hsCode HSCode
	result := s.db.WithContext(ctx).First(&hsCode, "id = ?", hsCodeID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("HS code with ID %s not found", hsCodeID)
		}
		return nil, fmt.Errorf("failed to retrieve HS code: %w", result.Error)
	}
	return &hsCode, nil
}

// GetByIDs retrieves multiple HS codes by their IDs from the database
func (s *Service) GetByIDs(ctx context.Context, hsCodeIDs []string) ([]HSCode, error) {
	if len(hsCodeIDs) == 0 {
		return []HSCode{}, nil
	}
	var hsCodes []HSCode
	result := s.db.WithContext(ctx).Where("id IN ?", hsCodeIDs).Find(&hsCodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve HS codes: %w", result.Error)
	}
	return hsCodes, nil
}

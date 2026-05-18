package hscode

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/utils"
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
	// Get total count first for pagination (with filter applied)
	var totalCount int64
	countQuery := s.db.WithContext(ctx).Model(&HSCode{})

	// Apply the same filter to the count query
	if filter.HSCodeStartsWith != nil && *filter.HSCodeStartsWith != "" {
		countQuery = countQuery.Where("hs_code LIKE ?", *filter.HSCodeStartsWith+"%")
	}

	countResult := countQuery.Count(&totalCount)
	if countResult.Error != nil {
		return nil, fmt.Errorf("failed to count HS codes: %w", countResult.Error)
	}

	// If no HS codes found, return early
	if totalCount == 0 {
		return &ListResult{
			TotalCount: 0,
			Items:      []HSCode{},
			Offset:     0,
			Limit:      0,
		}, nil
	}

	var hsCodes []HSCode
	query := s.db.WithContext(ctx)

	// Apply filter: HSCode starts with
	if filter.HSCodeStartsWith != nil && *filter.HSCodeStartsWith != "" {
		query = query.Where("hs_code LIKE ?", *filter.HSCodeStartsWith+"%")
	}

	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(filter.Offset, filter.Limit)
	query = query.Offset(finalOffset).Limit(finalLimit)

	// Add ordering for consistent pagination
	query = query.Order("hs_code ASC")

	result := query.Find(&hsCodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve HS codes: %w", result.Error)
	}

	// Prepare the result
	hsCodeListResult := &ListResult{
		TotalCount: totalCount,
		Items:      hsCodes,
		Offset:     finalOffset,
		Limit:      finalLimit,
	}

	return hsCodeListResult, nil
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

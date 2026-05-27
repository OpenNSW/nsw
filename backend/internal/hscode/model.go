package hscode

import (
	"time"

	"github.com/OpenNSW/nsw/backend/pkg/pagination"
)

// HSCode represents the Harmonized System Code used for classifying traded products.
type HSCode struct {
	ID          string    `gorm:"type:text;column:id;primaryKey;not null" json:"id"`
	HSCode      string    `gorm:"type:varchar(50);column:hs_code;not null;unique" json:"hsCode"`
	Description string    `gorm:"type:text;column:description" json:"description"`
	Category    string    `gorm:"type:text;column:category" json:"category"`
	CreatedAt   time.Time `gorm:"type:timestamptz;column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (h *HSCode) TableName() string {
	return "hs_codes"
}

// Filter will be used when querying as batch
type Filter struct {
	HSCodeStartsWith *string `json:"hsCodeStartsWith,omitempty"`
	Offset           *int    `json:"offset,omitempty"`
	Limit            *int    `json:"limit,omitempty"`
}

// ListResult is the pagination envelope returned by the list HS codes endpoint.
type ListResult = pagination.Page[HSCode]

// ResponseDTO represents HS Code details in the response.
type ResponseDTO struct {
	HSCodeID    string `json:"hsCodeId"`    // HS Code ID
	HSCode      string `json:"hsCode"`      // HS Code
	Description string `json:"description"` // Description of the HS Code
	Category    string `json:"category"`    // Category of the HS Code
}

package model

// HSCode represents the Harmonized System Code used for classifying traded products.
type HSCode struct {
	BaseModel
	HSCode      string `gorm:"type:varchar(50);column:hs_code;not null;unique" json:"hsCode"` // HS Code
	Description string `gorm:"type:text;column:description" json:"description"`               // Description of the HS Code
	Category    string `gorm:"type:text;column:category" json:"category"`                     // Category of the HS Code
}

func (h *HSCode) TableName() string {
	return "hs_codes"
}

// HSCodeFilter will be used when querying as batch
type HSCodeFilter struct {
	HSCodeStartsWith *string `json:"hsCodeStartsWith,omitempty"`
	Offset           *int    `json:"offset,omitempty"`
	Limit            *int    `json:"limit,omitempty"`
}

// HSCodeListResult represents the result of querying HS codes with pagination
type HSCodeListResult struct {
	TotalCount int64    `json:"totalCount"`
	HSCodes    []HSCode `json:"hsCodes"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

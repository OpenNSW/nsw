package model

// HSCode represents the Harmonized System Code used for classifying traded products.
type HSCode struct {
	BaseModel
	Code        string `gorm:"type:varchar(50);column:code;not null;unique" json:"code"` // HS Code
	Description string `gorm:"type:text;column:description" json:"description"`          // Description of the HS Code
}

func (h *HSCode) TableName() string {
	return "hs_codes"
}

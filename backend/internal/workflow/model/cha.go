package model

// ClearingHouseAgent represents a Customs House Agent (CHA) entity.
type ClearingHouseAgent struct {
	BaseModel
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
}

func (c *ClearingHouseAgent) TableName() string {
	return "clearing_house_agents"
}

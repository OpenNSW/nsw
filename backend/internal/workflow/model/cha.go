package model

// CustomsHouseAgent represents a CHA entity.
type CustomsHouseAgent struct {
	BaseModel
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
}

func (c *CustomsHouseAgent) TableName() string {
	return "customs_house_agents"
}

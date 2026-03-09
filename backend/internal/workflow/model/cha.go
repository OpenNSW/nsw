package model

// CHA (Customs House Agent) represents a clearing house agent that can be assigned to a consignment.
type CHA struct {
	BaseModel
	Name        string `gorm:"type:varchar(255);column:name;not null" json:"name"`
	Description string `gorm:"type:text;column:description" json:"description"`
}

func (c *CHA) TableName() string {
	return "customs_house_agents"
}

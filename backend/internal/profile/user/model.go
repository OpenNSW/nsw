package user

import (
	"encoding/json"
)

// Record represents a user's persisted profile in the database.
// This model intentionally excludes roles; roles are derived from the authentication principal.
type Record struct {
	UserID      string          `gorm:"type:varchar(100);column:user_id;primaryKey;not null" json:"userId"`
	Email       string          `gorm:"type:varchar(255);column:email" json:"email"`
	PhoneNumber string          `gorm:"type:varchar(20);column:phone_number" json:"phoneNumber"`
	OUID        string          `gorm:"type:varchar(255);column:ou_id" json:"ouId"`
	NSWData     json.RawMessage `gorm:"type:jsonb;column:nsw_data" json:"nswData"`
}

// TableName specifies the database table for this model.
func (r *Record) TableName() string {
	return "user_records"
}

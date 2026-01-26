package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel defines the base model structure with common fields for the form package.
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;column:id;not null;primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"type:timestamptz;column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:timestamptz;column:updated_at;not null" json:"updatedAt"`
}

// BeforeCreate is a GORM hook that is triggered before a new record is created.
func (base *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	base.ID, err = uuid.NewRandom()
	if err != nil {
		return
	}
	base.CreatedAt = time.Now().UTC()
	base.UpdatedAt = time.Now().UTC()
	return
}

// BeforeUpdate is a GORM hook that is triggered before an existing record is updated.
func (base *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	base.UpdatedAt = time.Now().UTC()
	return
}

// Form represents a form definition that can be rendered using JSON Forms
type Form struct {
	BaseModel
	Name        string          `gorm:"type:varchar(255);column:name;not null" json:"name"`                    // Human-readable form name
	Description string          `gorm:"type:text;column:description" json:"description,omitempty"`             // Optional description
	Schema      json.RawMessage `gorm:"type:jsonb;column:schema;not null" json:"schema"`                       // JSON Schema definition
	UISchema    json.RawMessage `gorm:"type:jsonb;column:ui_schema;not null" json:"uiSchema"`                  // UI Schema definition for JSON Forms
	Version     string          `gorm:"type:varchar(50);column:version;not null;default:'1.0'" json:"version"` // Form version
	Active      bool            `gorm:"type:boolean;column:active;not null;default:true" json:"active"`        // Whether this form is active
}

func (f *Form) TableName() string {
	return "forms"
}

// FormResponse represents the response structure for form retrieval
// This is what portals receive - they don't need to know about Task/FormType
type FormResponse struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	Schema   json.RawMessage `json:"schema"`   // JSON Schema
	UISchema json.RawMessage `json:"uiSchema"` // UI Schema
	Version  string          `json:"version"`
}

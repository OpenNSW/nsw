package preconsignment

import workflowmodel "github.com/OpenNSW/nsw/internal/workflow/model"

type PreConsignmentState string

const (
	PreConsignmentStateLocked     PreConsignmentState = "LOCKED"
	PreConsignmentStateReady      PreConsignmentState = "READY"
	PreConsignmentStateInProgress PreConsignmentState = "IN_PROGRESS"
	PreConsignmentStateCompleted  PreConsignmentState = "COMPLETED"
)

type PreConsignmentTemplate struct {
	workflowmodel.BaseModel
	Name               string   `gorm:"type:varchar(255);column:name;not null" json:"name"`
	Description        string   `gorm:"type:text;column:description" json:"description"`
	WorkflowTemplateID string   `json:"workflowTemplateId"`
	DependsOn          []string `gorm:"type:jsonb;column:depends_on;serializer:json" json:"dependsOn"`
}

func (pct *PreConsignmentTemplate) TableName() string {
	return "pre_consignment_templates"
}

type PreConsignment struct {
	workflowmodel.BaseModel
	TraderID                 string              `gorm:"type:varchar(255);not null" json:"traderId"`
	PreConsignmentTemplateID string              `gorm:"type:text;not null" json:"preConsignmentTemplateId"`
	State                    PreConsignmentState `gorm:"type:varchar(50);not null" json:"state"`

	PreConsignmentTemplate PreConsignmentTemplate  `gorm:"foreignKey:PreConsignmentTemplateID;references:ID" json:"-"`
	Workflow               *workflowmodel.Workflow `gorm:"foreignKey:ID;references:ID" json:"-"`
}

func (pc *PreConsignment) TableName() string {
	return "pre_consignments"
}

type CreatePreConsignmentDTO struct {
	PreConsignmentTemplateID string `json:"preConsignmentTemplateId" validate:"required"`
}

type UpdatePreConsignmentStateDTO struct {
	State         PreConsignmentState `json:"state" validate:"required"`
	TraderContext map[string]any      `json:"traderContext"`
}

type PreConsignmentTemplateResponseDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DependsOn   []string `json:"dependsOn"`
}

type TraderPreConsignmentResponseDTO struct {
	ID               string              `json:"id"`
	PreConsignmentID *string             `json:"preConsignmentId,omitempty"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	DependsOn        []string            `json:"dependsOn"`
	State            PreConsignmentState `json:"state"`

	PreConsignment *PreConsignment `json:"preConsignment,omitempty"`
}

type TraderPreConsignmentsResponseDTO struct {
	TotalCount int64                             `json:"totalCount"`
	Items      []TraderPreConsignmentResponseDTO `json:"items"`
	Offset     int64                             `json:"offset"`
	Limit      int64                             `json:"limit"`
}

type PreConsignmentResponseDTO struct {
	ID                     string                                  `json:"id"`
	TraderID               string                                  `json:"traderId"`
	State                  PreConsignmentState                     `json:"state"`
	TraderContext          map[string]any                          `json:"traderContext"`
	CreatedAt              string                                  `json:"createdAt"`
	UpdatedAt              string                                  `json:"updatedAt"`
	PreConsignmentTemplate PreConsignmentTemplateResponseDTO       `json:"preConsignmentTemplate"`
	WorkflowNodes          []workflowmodel.WorkflowNodeResponseDTO `json:"workflowNodes"`
}

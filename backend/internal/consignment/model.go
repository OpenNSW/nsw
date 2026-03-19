package consignment

import workflowmodel "github.com/OpenNSW/nsw/internal/workflow/model"

// ConsignmentFlow represents the flow type of a consignment.
type ConsignmentFlow string

const (
	ConsignmentFlowImport ConsignmentFlow = "IMPORT"
	ConsignmentFlowExport ConsignmentFlow = "EXPORT"
)

// ConsignmentState represents the state of a consignment.
type ConsignmentState string

const (
	ConsignmentStateInitialized ConsignmentState = "INITIALIZED"
	ConsignmentStateInProgress  ConsignmentState = "IN_PROGRESS"
	ConsignmentStateFinished    ConsignmentState = "FINISHED"
)

// Consignment represents a consignment in the system.
type Consignment struct {
	workflowmodel.BaseModel
	Flow     ConsignmentFlow   `gorm:"type:varchar(50);column:flow;not null" json:"flow"`
	TraderID string            `gorm:"type:varchar(100);column:trader_id;not null" json:"traderId"`
	State    ConsignmentState  `gorm:"type:varchar(50);column:state;not null" json:"state"`
	Items    []ConsignmentItem `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"`

	// CHA (Customs House Agent) – set at Stage 1 by Trader; CHA completes Stage 2 by selecting HS Code.
	CHAID *string `gorm:"type:text;column:cha_id" json:"chaId,omitempty"`
	CHA   *CHA    `gorm:"foreignKey:CHAID" json:"cha,omitempty"`

	// Relationships.
	Workflow *workflowmodel.Workflow `gorm:"foreignKey:ID;references:ID" json:"-"`
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// CHA (Customs House Agent) represents a clearing house agent that can be assigned to a consignment.
type CHA struct {
	workflowmodel.BaseModel
	Name        string `gorm:"type:varchar(255);column:name;not null" json:"name"`
	Description string `gorm:"type:text;column:description" json:"description"`
	Email       string `gorm:"type:varchar(255);column:email" json:"email,omitempty"`
}

func (c *CHA) TableName() string {
	return "customs_house_agents"
}

// HSCode represents the Harmonized System Code used for classifying traded products.
type HSCode struct {
	workflowmodel.BaseModel
	HSCode      string `gorm:"type:varchar(50);column:hs_code;not null;unique" json:"hsCode"`
	Description string `gorm:"type:text;column:description" json:"description"`
	Category    string `gorm:"type:text;column:category" json:"category"`
}

func (h *HSCode) TableName() string {
	return "hs_codes"
}

// HSCodeFilter will be used when querying as batch.
type HSCodeFilter struct {
	HSCodeStartsWith *string `json:"hsCodeStartsWith,omitempty"`
	Offset           *int    `json:"offset,omitempty"`
	Limit            *int    `json:"limit,omitempty"`
}

// HSCodeListResult represents the result of querying HS codes with pagination.
type HSCodeListResult struct {
	TotalCount int64    `json:"totalCount"`
	Items      []HSCode `json:"items"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

// ConsignmentItem represents an individual item within a consignment.
type ConsignmentItem struct {
	HSCodeID string `gorm:"type:text;column:hs_code_id;not null" json:"hsCodeId"`
}

// ConsignmentItemResponseDTO represents an individual item in the consignment response.
type ConsignmentItemResponseDTO struct {
	HSCode HSCodeResponseDTO `json:"hsCode"`
}

// HSCodeResponseDTO represents HS Code details in the response.
type HSCodeResponseDTO struct {
	HSCodeID    string `json:"hsCodeId"`
	HSCode      string `json:"hsCode"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// InitializeConsignmentDTO is the request body for PUT /consignments/{id}/initialize.
type InitializeConsignmentDTO struct {
	HSCodeIDs []string `json:"hsCodeIds" binding:"required,min=1"`
}

// CreateConsignmentItemDTO represents the data required to create a consignment item.
type CreateConsignmentItemDTO struct {
	HSCodeID string `json:"hsCodeId" binding:"required"`
}

// CreateConsignmentDTO represents the data required to create a consignment.
type CreateConsignmentDTO struct {
	Flow  ConsignmentFlow            `json:"flow" binding:"required,oneof=IMPORT EXPORT"`
	ChaID *string                    `json:"chaId,omitempty"`
	Items []CreateConsignmentItemDTO `json:"items,omitempty"`
}

// UpdateConsignmentDTO represents the data required to update a consignment.
type UpdateConsignmentDTO struct {
	ConsignmentID         string            `json:"consignmentId" binding:"required"`
	State                 *ConsignmentState `json:"state,omitempty"`
	AppendToGlobalContext map[string]any    `json:"appendToGlobalContext,omitempty"`
}

// ConsignmentDetailDTO represents the full consignment data returned in detailed responses.
type ConsignmentDetailDTO struct {
	ID            string                                  `json:"id"`
	Flow          ConsignmentFlow                         `json:"flow"`
	TraderID      string                                  `json:"traderId"`
	ChaID         *string                                 `json:"chaId,omitempty"`
	State         ConsignmentState                        `json:"state"`
	Items         []ConsignmentItemResponseDTO            `json:"items"`
	CreatedAt     string                                  `json:"createdAt"`
	UpdatedAt     string                                  `json:"updatedAt"`
	WorkflowNodes []workflowmodel.WorkflowNodeResponseDTO `json:"workflowNodes"`
}

// ConsignmentSummaryDTO represents the consignment data returned in list responses.
type ConsignmentSummaryDTO struct {
	ID                         string                       `json:"id"`
	Flow                       ConsignmentFlow              `json:"flow"`
	TraderID                   string                       `json:"traderId"`
	ChaID                      *string                      `json:"chaId,omitempty"`
	State                      ConsignmentState             `json:"state"`
	Items                      []ConsignmentItemResponseDTO `json:"items"`
	CreatedAt                  string                       `json:"createdAt"`
	UpdatedAt                  string                       `json:"updatedAt"`
	WorkflowNodeCount          int                          `json:"workflowNodeCount"`
	CompletedWorkflowNodeCount int                          `json:"completedWorkflowNodeCount"`
}

// ConsignmentListResult represents the result of querying consignments with pagination.
type ConsignmentListResult struct {
	TotalCount int64                   `json:"totalCount"`
	Items      []ConsignmentSummaryDTO `json:"items"`
	Offset     int                     `json:"offset"`
	Limit      int                     `json:"limit"`
}

// ConsignmentFilter will be used when querying consignments as batch.
type ConsignmentFilter struct {
	TraderID *string           `json:"traderId,omitempty"`
	ChaID    *string           `json:"chaId,omitempty"`
	Flow     *ConsignmentFlow  `json:"flow,omitempty"`
	State    *ConsignmentState `json:"state,omitempty"`
	Offset   *int              `json:"offset,omitempty"`
	Limit    *int              `json:"limit,omitempty"`
}

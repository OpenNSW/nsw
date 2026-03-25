package model

type WorkflowTemplate struct {
	BaseModel
	Name              string      `gorm:"type:varchar(100);column:name;not null" json:"name"`                       // Name of the workflow template
	Description       string      `gorm:"type:text;column:description" json:"description"`                          // Description of the workflow template
	Version           string      `gorm:"type:varchar(50);column:version;not null" json:"version"`                  // Version of the workflow template
	NodeTemplates     StringArray `gorm:"type:jsonb;column:nodes;not null;serializer:json" json:"nodes"`            // Array of workflow node template IDs
	EndNodeTemplateID *string     `gorm:"type:text;column:end_node_template_id" json:"endNodeTemplateId,omitempty"` // Optional end node template ID. If set, workflow is complete when this node is completed.
}

func (wt *WorkflowTemplate) TableName() string {
	return "workflow_templates"
}

func (wt *WorkflowTemplate) GetNodeTemplateIDs() []string {
	return wt.NodeTemplates
}

type WorkflowTemplateV2 struct {
	BaseModel
	Name    string `gorm:"type:varchar(100);column:name;not null" json:"name"`
	Version int    `gorm:"type:integer;column:version;not null" json:"version"`
	Nodes   []Node `gorm:"type:jsonb;column:nodes;not null;serializer:json" json:"nodes"`
	Edges   []Edge `gorm:"type:jsonb;column:edges;not null;serializer:json" json:"edges"`
}

func (wt *WorkflowTemplateV2) TableName() string {
	return "workflow_template_v2"
}

type NodeType string

const (
	NodeTypeStart   NodeType = "START"
	NodeTypeEnd     NodeType = "END"
	NodeTypeTask    NodeType = "TASK"
	NodeTypeGateway NodeType = "GATEWAY"
)

type GatewayType string

const (
	GatewayTypeExclusiveSplit GatewayType = "EXCLUSIVE_SPLIT" // XOR Split
	GatewayTypeParallelSplit  GatewayType = "PARALLEL_SPLIT"  // AND Split
	GatewayTypeExclusiveJoin  GatewayType = "EXCLUSIVE_JOIN"  // XOR Join
	GatewayTypeParallelJoin   GatewayType = "PARALLEL_JOIN"   // AND Join
)

// Node represents a step in the workflow graph.
type Node struct {
	ID             string            `json:"id"`
	Type           NodeType          `json:"type"`                       // START, END, TASK, or GATEWAY
	GatewayType    GatewayType       `json:"gateway_type,omitempty"`     // See Gateway Types constants
	TaskTemplateID string            `json:"task_template_id,omitempty"` // Identifier for the task template to run
	OutputMapping  map[string]string `json:"output_mapping,omitempty"`   // Maps Task Output Key -> WorkflowVariables Key
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	TargetID  string `json:"target_id"`
	Condition string `json:"condition,omitempty"` // Expression mapped against WorkflowVariables
}

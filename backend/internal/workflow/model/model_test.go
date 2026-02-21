package model

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUIDArray_MarshalJSON(t *testing.T) {
	u1 := uuid.New()
	u2 := uuid.New()

	tests := []struct {
		name     string
		input    UUIDArray
		expected string
	}{
		{
			name:     "Populated",
			input:    UUIDArray{u1, u2},
			expected: `["` + u1.String() + `","` + u2.String() + `"]`,
		},
		{
			name:     "Empty",
			input:    UUIDArray{},
			expected: `[]`,
		},
		{
			name:     "Nil",
			input:    nil,
			expected: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestUUIDArray_UnmarshalJSON(t *testing.T) {
	u1 := uuid.New()
	u2 := uuid.New()

	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  UUIDArray
	}{
		{
			name:      "Valid Populated",
			input:     `["` + u1.String() + `","` + u2.String() + `"]`,
			expectErr: false,
			expected:  UUIDArray{u1, u2},
		},
		{
			name:      "Valid Empty",
			input:     `[]`,
			expectErr: false,
			expected:  UUIDArray{},
		},
		{
			name:      "Null",
			input:     `null`,
			expectErr: false,
			expected:  UUIDArray{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result UUIDArray
			err := json.Unmarshal([]byte(tt.input), &result)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestModel_TableNames(t *testing.T) {
	assert.Equal(t, "consignments", (&Consignment{}).TableName())
	assert.Equal(t, "hs_codes", (&HSCode{}).TableName())
	assert.Equal(t, "pre_consignment_templates", (&PreConsignmentTemplate{}).TableName())
	assert.Equal(t, "pre_consignments", (&PreConsignment{}).TableName())
	assert.Equal(t, "workflow_templates", (&WorkflowTemplate{}).TableName())
	assert.Equal(t, "workflow_node_templates", (&WorkflowNodeTemplate{}).TableName())
	assert.Equal(t, "workflow_nodes", (&WorkflowNode{}).TableName())
	assert.Equal(t, "workflow_template_maps", (&WorkflowTemplateMap{}).TableName())
}

func TestModel_WorkflowTemplate_GetNodeTemplateIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	wt := &WorkflowTemplate{
		NodeTemplates: UUIDArray{id1, id2},
	}
	ids := wt.GetNodeTemplateIDs()
	assert.Len(t, ids, 2)
	assert.ElementsMatch(t, []uuid.UUID{id1, id2}, ids)
}

func TestModel_BaseHooks(t *testing.T) {
	base := &BaseModel{}
	err := base.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotNil(t, base.ID)
	assert.False(t, base.CreatedAt.IsZero())
	assert.False(t, base.UpdatedAt.IsZero())

	err = base.BeforeUpdate(nil)
	assert.NoError(t, err)
}

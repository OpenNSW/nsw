package store

import (
	"encoding/json"
	"fmt"

	tfstore "github.com/OpenNSW/nsw-task-flow/store"
)

func toRow(r tfstore.TaskRecord) (taskRow, error) {
	dataMap := r.Data
	if dataMap == nil {
		dataMap = map[string]any{}
	}
	dataBytes, err := json.Marshal(dataMap)
	if err != nil {
		return taskRow{}, fmt.Errorf("marshal data: %w", err)
	}
	return taskRow{
		TaskID:               r.TaskID,
		TaskType:             r.TaskType,
		UserFormID:           r.UserFormID,
		ReviewerFormID:       r.ReviewerFormID,
		Status:               r.Status,
		ParentWorkflowID:     r.ParentWorkflowID,
		ParentRunID:          r.ParentRunID,
		ParentNodeID:         r.ParentNodeID,
		TaskWorkflowID:       r.TaskWorkflowID,
		TaskRunID:            r.TaskRunID,
		SubTaskNodeID:        r.SubTaskNodeID,
		ActiveTaskTemplateID: r.ActiveTaskTemplateID,
		Data:                 jsonbBytes(dataBytes),
		CreatedAt:            r.CreatedAt,
	}, nil
}

func fromRow(row taskRow) (tfstore.TaskRecord, error) {
	rec := tfstore.TaskRecord{
		TaskID:               row.TaskID,
		TaskType:             row.TaskType,
		UserFormID:           row.UserFormID,
		ReviewerFormID:       row.ReviewerFormID,
		Status:               row.Status,
		ParentWorkflowID:     row.ParentWorkflowID,
		ParentRunID:          row.ParentRunID,
		ParentNodeID:         row.ParentNodeID,
		TaskWorkflowID:       row.TaskWorkflowID,
		TaskRunID:            row.TaskRunID,
		SubTaskNodeID:        row.SubTaskNodeID,
		ActiveTaskTemplateID: row.ActiveTaskTemplateID,
		CreatedAt:            row.CreatedAt,
	}
	if len(row.Data) == 0 {
		rec.Data = map[string]any{}
		return rec, nil
	}
	var data map[string]any
	if err := json.Unmarshal(row.Data, &data); err != nil {
		return tfstore.TaskRecord{}, fmt.Errorf("unmarshal data: %w", err)
	}
	rec.Data = data
	return rec, nil
}

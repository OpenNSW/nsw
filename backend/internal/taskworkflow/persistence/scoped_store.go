package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

// TaskScopedRecord is the subset of a task workflow row exposed to a single
// task through TaskScopedStore.
type TaskScopedRecord struct {
	TaskID string          `json:"taskId"`
	State  plugin.State    `json:"state"`
	Data   json.RawMessage `json:"data"`
}

// TaskScopedStore restricts persistence access to one task row.
//
// It is intended for subtasks, activities, plugins, or task render code that
// should only read/update the current task's state and render data. The task ID
// is bound by NewTaskScopedStore, so callers cannot target another task.
type TaskScopedStore interface {
	Get() (*TaskScopedRecord, error)
	GetState() (plugin.State, error)
	SetState(plugin.State) error
	GetData() (json.RawMessage, error)
	SetData(json.RawMessage) error
}

type taskScopedStore struct {
	store  Store
	taskID string
}

func NewTaskScopedStore(store Store, taskID string) (TaskScopedStore, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	return &taskScopedStore{
		store:  store,
		taskID: taskID,
	}, nil
}

func (s *taskScopedStore) Get() (*TaskScopedRecord, error) {
	task, err := s.store.GetByTaskID(s.taskID)
	if err != nil {
		return nil, err
	}

	return &TaskScopedRecord{
		TaskID: task.TaskID,
		State:  task.State,
		Data:   task.Data,
	}, nil
}

func (s *taskScopedStore) GetState() (plugin.State, error) {
	return s.store.GetStateByTaskID(s.taskID)
}

func (s *taskScopedStore) SetState(state plugin.State) error {
	return s.store.UpdateState(s.taskID, state)
}

func (s *taskScopedStore) GetData() (json.RawMessage, error) {
	return s.store.GetDataByTaskID(s.taskID)
}

func (s *taskScopedStore) SetData(data json.RawMessage) error {
	return s.store.UpdateData(s.taskID, data)
}

package task

// Type represents the type of task
type Type string

const (
	TaskTypeSimpleForm   Type = "SIMPLE_FORM"
	TaskTypeWaitForEvent Type = "WAIT_FOR_EVENT"
)

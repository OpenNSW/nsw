package plugin

// Type represents the type of task
type Type string

const (
	TaskTypeSimpleForm   Type = "SIMPLE_FORM"
	TaskTypeWaitForEvent Type = "WAIT_FOR_EVENT"
)

type State string

const (
	InProgress State = "IN_PROGRESS"
	Completed  State = "COMPLETED"
	Failed     State = "FAILED"
)

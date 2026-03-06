package plugin

// Type represents the type of task
type Type string

const (
	TaskTypeSimpleForm   Type = "SIMPLE_FORM"
	TaskTypeWaitForEvent Type = "WAIT_FOR_EVENT"
	TaskTypePayment      Type = "PAYMENT"
)

type State string

const (
	Initialized State = "INITIALIZED"
	InProgress  State = "IN_PROGRESS"
	Completed   State = "COMPLETED"
	Failed      State = "FAILED"
)

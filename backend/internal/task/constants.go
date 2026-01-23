package task

// Type represents the type of task
type Type string

const (
	TaskTypeTraderForm     Type = "TRADER_FORM"
	TaskTypeOGAForm        Type = "OGA_FORM"
	TaskTypeWaitForEvent   Type = "WAIT_FOR_EVENT"
	TaskTypeDocumentSubmit Type = "DOCUMENT_SUBMISSION" // Step for document submission
	TaskTypePayment        Type = "PAYMENT"
)

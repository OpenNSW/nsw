package internal

import "time"

const (
	// DefaultPresignTTL is the default time-to-live for presigned upload and download URLs
	DefaultPresignTTL = 15 * time.Minute

	// Application Statuses
	StatusPending           = "PENDING"
	StatusApproved          = "APPROVED"
	StatusRejected          = "REJECTED"
	StatusFeedbackRequested = "FEEDBACK_REQUESTED"
)

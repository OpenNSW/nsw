package oga

import "github.com/google/uuid"

// OGATaskNotification represents the notification sent to OGA service
type OGATaskNotification struct {
	TaskID        uuid.UUID `json:"taskId"`
	ConsignmentID uuid.UUID `json:"consignmentId"`
	FormID        string    `json:"formId"`
	Status        string    `json:"status"`
}

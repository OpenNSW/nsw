package notifications

import (
	"fmt"
)

type ChannelType string

const (
	ChannelSMS   ChannelType = "sms"
	ChannelEmail ChannelType = "email"
)

type Request struct {
	Channel  ChannelType
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

func (r Request) Validate() error {
	switch r.Channel {
	case ChannelSMS, ChannelEmail:
		// valid
	default:
		return fmt.Errorf("invalid or missing channel: %q", r.Channel)
	}
	if r.To == "" {
		return fmt.Errorf("to is required")
	}
	if r.Body == "" && r.HTMLBody == "" {
		return fmt.Errorf("body or html_body is required")
	}
	return nil
}

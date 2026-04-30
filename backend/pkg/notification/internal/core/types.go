package core

import (
	"context"
	"encoding/json"
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
	if r.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if r.To == "" {
		return fmt.Errorf("to is required")
	}
	if r.Body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

type Provider interface {
	Type() ChannelType
	Configure(cfg json.RawMessage) error
	Send(ctx context.Context, req Request) error
}

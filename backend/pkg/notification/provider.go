package notification

import (
	"context"
	"encoding/json"
)

type Provider interface {
	Type() ChannelType
	Configure(cfg json.RawMessage) error
	Send(ctx context.Context, req Request) error
}

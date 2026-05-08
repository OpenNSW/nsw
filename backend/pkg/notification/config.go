package notification

import "fmt"

type Config struct {
	Path string
}

func (c Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("NOTIFICATION_CONFIG_PATH is required")
	}
	return nil
}

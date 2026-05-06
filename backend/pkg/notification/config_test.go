package notification

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "missing path",
			config:  Config{},
			wantErr: true,
		},
		{
			name:    "valid path",
			config:  Config{Path: "/tmp/notification.json"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != "NOTIFICATION_CONFIG_PATH is required" {
					t.Errorf("error = %q, want %q", err.Error(), "NOTIFICATION_CONFIG_PATH is required")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

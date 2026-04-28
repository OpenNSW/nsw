package temporal

import (
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
)

func TestOptionsFromConfigDefaults(t *testing.T) {
	opts := optionsFromConfig(config.TemporalConfig{})

	if opts.HostPort != "localhost:7233" {
		t.Fatalf("HostPort default = %q, want %q", opts.HostPort, "localhost:7233")
	}
	if opts.Namespace != "default" {
		t.Fatalf("Namespace default = %q, want %q", opts.Namespace, "default")
	}
}

func TestOptionsFromConfigOverrides(t *testing.T) {
	cfg := config.TemporalConfig{
		HostPort:  " temporal.example:7233 ",
		Namespace: " staging ",
	}
	opts := optionsFromConfig(cfg)

	if opts.HostPort != "temporal.example:7233" {
		t.Fatalf("HostPort override = %q, want %q", opts.HostPort, "temporal.example:7233")
	}
	if opts.Namespace != "staging" {
		t.Fatalf("Namespace override = %q, want %q", opts.Namespace, "staging")
	}
}

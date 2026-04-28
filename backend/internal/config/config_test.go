package config

import "testing"

func TestLoadTemporalDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("TEMPORAL_HOST_PORT", "")
	t.Setenv("TEMPORAL_NAMESPACE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Temporal.HostPort != "localhost:7233" {
		t.Fatalf("HostPort default = %q, want %q", cfg.Temporal.HostPort, "localhost:7233")
	}
	if cfg.Temporal.Namespace != "default" {
		t.Fatalf("Namespace default = %q, want %q", cfg.Temporal.Namespace, "default")
	}
}

func TestLoadTemporalOverrides(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("TEMPORAL_HOST_PORT", " temporal.example:7233 ")
	t.Setenv("TEMPORAL_NAMESPACE", " staging ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Temporal.HostPort != "temporal.example:7233" {
		t.Fatalf("HostPort override = %q, want %q", cfg.Temporal.HostPort, "temporal.example:7233")
	}
	if cfg.Temporal.Namespace != "staging" {
		t.Fatalf("Namespace override = %q, want %q", cfg.Temporal.Namespace, "staging")
	}
}

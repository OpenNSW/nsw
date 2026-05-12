package config

import (
	"testing"
)

func TestLoadTemporalDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("TEMPORAL_HOST", "")
	t.Setenv("TEMPORAL_PORT", "")
	t.Setenv("TEMPORAL_NAMESPACE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Temporal.Host != "localhost" {
		t.Fatalf("Host default = %q, want %q", cfg.Temporal.Host, "localhost")
	}
	if cfg.Temporal.Port != 7233 {
		t.Fatalf("Port default = %d, want %d", cfg.Temporal.Port, 7233)
	}
	if cfg.Temporal.Namespace != "default" {
		t.Fatalf("Namespace default = %q, want %q", cfg.Temporal.Namespace, "default")
	}
}

func TestLoadTemporalOverrides(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("TEMPORAL_HOST", " temporal.example ")
	t.Setenv("TEMPORAL_PORT", " 7234 ")
	t.Setenv("TEMPORAL_NAMESPACE", " staging ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Temporal.Host != "temporal.example" {
		t.Fatalf("Host override = %q, want %q", cfg.Temporal.Host, "temporal.example")
	}
	if cfg.Temporal.Port != 7234 {
		t.Fatalf("Port override = %d, want %d", cfg.Temporal.Port, 7234)
	}
	if cfg.Temporal.Namespace != "staging" {
		t.Fatalf("Namespace override = %q, want %q", cfg.Temporal.Namespace, "staging")
	}
}

func TestLoadTemporalInvalidPortFallsBackToDefault(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("TEMPORAL_PORT", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Temporal.Port != 7233 {
		t.Fatalf("Port = %d, want default %d", cfg.Temporal.Port, 7233)
	}
}

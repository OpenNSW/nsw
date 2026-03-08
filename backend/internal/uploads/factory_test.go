package uploads

import (
	"context"
	"os"
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
)

func TestFactory_New_Local(t *testing.T) {
	dir, err := os.MkdirTemp("", "uploads-factory-local")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	f := NewFactory()
	cfg := config.StorageConfig{
		Type:           "local",
		LocalBaseDir:   dir,
		LocalPublicURL: "http://localhost:8080/uploads",
	}
	driver, err := f.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New(local): %v", err)
	}
	if driver == nil {
		t.Fatal("expected non-nil driver")
	}
}

func TestFactory_New_InvalidType(t *testing.T) {
	f := NewFactory()
	cfg := config.StorageConfig{Type: "invalid"}
	driver, err := f.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if driver != nil {
		t.Fatal("expected nil driver")
	}
}

func TestFactory_New_LocalTrimSpace(t *testing.T) {
	dir, err := os.MkdirTemp("", "uploads-factory-local")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	f := NewFactory()
	cfg := config.StorageConfig{
		Type:           " local ", // with spaces; config loader trims
		LocalBaseDir:   dir,
		LocalPublicURL: "/uploads",
	}
	// Factory trims cfg.Type internally
	driver, err := f.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New(local with spaces): %v", err)
	}
	if driver == nil {
		t.Fatal("expected non-nil driver")
	}
}

package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "notifications-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

const exampleConfig = `{
  "channels": [
    {
      "provider": "email_service",
      "emailTemplateRoot": "./configs/email-templates",
      "options": {
        "baseURL": "https://email.svc.local",
        "token": "${EMAIL_SERVICE_TOKEN}"
      }
    },
    {
      "provider": "govsms",
      "smsTemplateRoot": "./configs/sms-templates",
      "options": {
        "baseURL": "https://govsms.example/api",
        "userName": "${GOVSMS_USERNAME}",
        "password": "${GOVSMS_PASSWORD}",
        "sidCode": "${GOVSMS_SID_CODE}"
      }
    }
  ]
}`

func TestLoadFromFile_ParsesExample(t *testing.T) {
	t.Setenv("EMAIL_SERVICE_TOKEN", "tok")
	t.Setenv("GOVSMS_USERNAME", "user")
	t.Setenv("GOVSMS_PASSWORD", "pass")
	t.Setenv("GOVSMS_SID_CODE", "sid")

	path := writeFixture(t, exampleConfig)
	m, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
}

func TestLoadFromFile_MissingEnvVar(t *testing.T) {
	os.Unsetenv("EMAIL_SERVICE_TOKEN")
	t.Setenv("GOVSMS_USERNAME", "user")
	t.Setenv("GOVSMS_PASSWORD", "pass")
	t.Setenv("GOVSMS_SID_CODE", "sid")

	path := writeFixture(t, exampleConfig)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "EMAIL_SERVICE_TOKEN") {
		t.Errorf("error should mention missing var, got: %v", err)
	}
}

func TestLoadFromFile_UnknownProvider(t *testing.T) {
	path := writeFixture(t, `{"channels":[{"provider":"slack","options":{}}]}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "slack") {
		t.Errorf("error should mention provider name, got: %v", err)
	}
}

func TestLoadFromFile_BadOptions_Email(t *testing.T) {
	t.Setenv("EMAIL_SERVICE_TOKEN", "tok")
	path := writeFixture(t, `{
		"channels":[{
			"provider":"email_service",
			"options":{"token":"${EMAIL_SERVICE_TOKEN}"}
		}]
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing baseURL")
	}
	if !strings.Contains(err.Error(), "baseURL") {
		t.Errorf("error should mention baseURL, got: %v", err)
	}
}

func TestLoadFromFile_BadOptions_GovSMS(t *testing.T) {
	path := writeFixture(t, `{
		"channels":[{
			"provider":"govsms",
			"options":{"baseURL":"https://govsms.example/api","password":"p","sidCode":"s"}
		}]
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing userName")
	}
	if !strings.Contains(err.Error(), "userName") {
		t.Errorf("error should mention userName, got: %v", err)
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFromFile_EmptyPath(t *testing.T) {
	_, err := LoadFromFile("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestExpandEnv_InjectionSafe(t *testing.T) {
	// A value containing JSON-special chars must not break the surrounding structure.
	t.Setenv("EVIL_TOKEN", `foo","injected":"bar`)
	input := []byte(`{"options":{"token":"${EVIL_TOKEN}"}}`)
	out, err := expandEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]map[string]string
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("expansion produced invalid JSON: %v\noutput: %s", err, out)
	}
	if parsed["options"]["token"] != `foo","injected":"bar` {
		t.Errorf("value mangled: %q", parsed["options"]["token"])
	}
	if _, injected := parsed["injected"]; injected {
		t.Error("injection succeeded: extra key found in parsed output")
	}
}

func TestExpandEnv_MultilinePEM(t *testing.T) {
	pem := "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkq\n-----END CERTIFICATE-----"
	t.Setenv("TLS_CERT", pem)
	input := []byte(`{"options":{"cert":"${TLS_CERT}"}}`)
	out, err := expandEnv(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]map[string]string
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("expansion produced invalid JSON for PEM value: %v\noutput: %s", err, out)
	}
	if parsed["options"]["cert"] != pem {
		t.Errorf("PEM value mangled: %q", parsed["options"]["cert"])
	}
}

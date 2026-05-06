package notification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnv_NoPlaceholders(t *testing.T) {
	data := []byte(`{"key": "value"}`)
	result, err := expandEnv(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(data) {
		t.Errorf("result = %q, want %q", string(result), string(data))
	}
}

func TestExpandEnv_SinglePlaceholder(t *testing.T) {
	os.Setenv("TEST_VAR", "replaced-value")
	defer os.Unsetenv("TEST_VAR")

	data := []byte(`{"key": "${TEST_VAR}"}`)
	result, err := expandEnv(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed["key"] != "replaced-value" {
		t.Errorf("key = %q, want %q", parsed["key"], "replaced-value")
	}
}

func TestExpandEnv_MultiplePlaceholders(t *testing.T) {
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")
	defer os.Unsetenv("VAR1")
	defer os.Unsetenv("VAR2")

	data := []byte(`{"a": "${VAR1}", "b": "${VAR2}"}`)
	result, err := expandEnv(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed["a"] != "value1" {
		t.Errorf("a = %q, want %q", parsed["a"], "value1")
	}
	if parsed["b"] != "value2" {
		t.Errorf("b = %q, want %q", parsed["b"], "value2")
	}
}

func TestExpandEnv_MissingVariable(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR")

	data := []byte(`{"key": "${NONEXISTENT_VAR}"}`)
	_, err := expandEnv(data)
	if err == nil {
		t.Fatal("expected error for missing variable, got nil")
	}
	if !contains(err.Error(), "NONEXISTENT_VAR") {
		t.Errorf("error %q does not mention NONEXISTENT_VAR", err.Error())
	}
}

func TestExpandEnv_MultipleUnsetVariables(t *testing.T) {
	os.Unsetenv("MISSING1")
	os.Unsetenv("MISSING2")

	data := []byte(`{"a": "${MISSING1}", "b": "${MISSING2}"}`)
	_, err := expandEnv(data)
	if err == nil {
		t.Fatal("expected error for missing variables, got nil")
	}

	errStr := err.Error()
	if !contains(errStr, "MISSING1") {
		t.Errorf("error %q does not mention MISSING1", errStr)
	}
	if !contains(errStr, "MISSING2") {
		t.Errorf("error %q does not mention MISSING2", errStr)
	}
}

func TestExpandEnv_SpecialCharactersInValue(t *testing.T) {
	os.Setenv("SPECIAL", `value with "quotes" and \backslash`)
	defer os.Unsetenv("SPECIAL")

	data := []byte(`{"key": "${SPECIAL}"}`)
	result, err := expandEnv(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed["key"] != `value with "quotes" and \backslash` {
		t.Errorf("key = %q, want %q", parsed["key"], `value with "quotes" and \backslash`)
	}
}

func TestLoadConfigMap_Success(t *testing.T) {
	os.Setenv("BASE_URL", "https://example.com")
	defer os.Unsetenv("BASE_URL")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	configData := []byte(`{
		"email": {"baseURL": "${BASE_URL}", "token": "secret"},
		"sms": {"baseURL": "https://sms.example.com"}
	}`)

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfgMap, err := loadConfigMap(configFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfgMap == nil {
		t.Fatal("cfgMap is nil")
	}

	if _, ok := cfgMap["email"]; !ok {
		t.Error("email key not found in config map")
	}
	if _, ok := cfgMap["sms"]; !ok {
		t.Error("sms key not found in config map")
	}
}

func TestLoadConfigMap_FileNotFound(t *testing.T) {
	_, err := loadConfigMap("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !contains(err.Error(), "read notification config") {
		t.Errorf("error message should mention 'read notification config', got: %v", err)
	}
}

func TestLoadConfigMap_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := loadConfigMap(configFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !contains(err.Error(), "parse notification config") {
		t.Errorf("error message should mention 'parse notification config', got: %v", err)
	}
}

func TestLoadConfigMap_MissingEnvVar(t *testing.T) {
	os.Unsetenv("MISSING_ENV_VAR")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	configData := []byte(`{"email": {"baseURL": "${MISSING_ENV_VAR}"}}`)
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := loadConfigMap(configFile)
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
	if !contains(err.Error(), "MISSING_ENV_VAR") {
		t.Errorf("error should mention MISSING_ENV_VAR, got: %v", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

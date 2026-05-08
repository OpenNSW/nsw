package notification

import (
	"encoding/json"
	"fmt"
	"os"
)

// expandEnv expands ${VAR_NAME} placeholders in data with their environment variable values.
// Values are JSON-escaped before substitution so special characters cannot corrupt the JSON document.
// Returns an error listing all unset variables if any are missing.
func expandEnv(data []byte) ([]byte, error) {
	var missing []string
	expanded := os.Expand(string(data), func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			missing = append(missing, key)
			return "${" + key + "}"
		}
		// JSON-escape the value; the template wraps ${VAR} in quotes so we strip them here.
		// json.Marshal on a string never returns an error.
		encoded, _ := json.Marshal(val)
		return string(encoded[1 : len(encoded)-1])
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("unset environment variables in notification config: %v", missing)
	}
	return []byte(expanded), nil
}

// loadConfigMap reads a JSON configuration file from the specified path, expands any
// environment variable placeholders, and unmarshals the result into a map.
//
// The function performs three main steps:
//  1. Reads the entire file contents
//  2. Expands all ${VAR_NAME} environment variable placeholders
//  3. Parses the resulting JSON into a map with string keys and raw JSON values
//
// This allows configuration files to reference environment variables which are
// substituted at runtime, enabling flexible deployment configurations.
//
// Parameters:
//   - path: The file system path to the notification configuration file
//
// Returns:
//   - A map where keys are provider channel names (strings) and values are raw JSON messages
//   - An error if the file cannot be read, environment variables cannot be expanded,
//     or the JSON is malformed
func loadConfigMap(path string) (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read notification config %q: %w", path, err)
	}
	expanded, err := expandEnv(raw)
	if err != nil {
		return nil, err
	}
	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(expanded, &cfgMap); err != nil {
		return nil, fmt.Errorf("parse notification config: %w", err)
	}
	return cfgMap, nil
}

package notification

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// envVarRe is a regex pattern that matches environment variable placeholders in the format ${VAR_NAME}.
// It captures the variable name without the ${} delimiters.
var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnv processes byte data to expand environment variable placeholders.
// It looks for patterns like ${VAR_NAME} and replaces them with their corresponding
// environment variable values. The replacement value is JSON-encoded to ensure proper
// escaping when embedded in JSON configuration files.
//
// If any environment variables referenced in the data are not set, expandEnv returns
// an error listing all missing variables and leaves those placeholders unchanged.
//
// Parameters:
//   - data: The byte slice containing potential environment variable placeholders
//
// Returns:
//   - A byte slice with all environment variables expanded
//   - An error if any referenced environment variables are not set
func expandEnv(data []byte) ([]byte, error) {
	var missing []string
	result := envVarRe.ReplaceAllFunc(data, func(match []byte) []byte {
		// Extract the variable name by removing ${} wrapper
		// match[2 : len(match)-1] skips the first 2 chars (${ ) and last 1 char (})
		name := string(match[2 : len(match)-1])
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		// JSON encode the value to ensure proper escaping, then strip the surrounding quotes
		encoded, _ := json.Marshal(val)
		return encoded[1 : len(encoded)-1]
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("unset environment variables in notification config: %v", missing)
	}
	return result, nil
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

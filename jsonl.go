package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// ParseJSONLFile reads a JSONL file and returns pretty-printed JSON content
// with nested JSON strings expanded
func ParseJSONLFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var data interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			// If parsing fails, just include the raw line
			if !first {
				result.WriteString("\n\n")
			}
			result.WriteString(line)
			first = false
			continue
		}

		// Process nested JSON strings
		processed := processValue(data)

		// Pretty print with 4-space indentation
		pretty, err := json.MarshalIndent(processed, "", "    ")
		if err != nil {
			if !first {
				result.WriteString("\n\n")
			}
			result.WriteString(line)
			first = false
			continue
		}

		if !first {
			result.WriteString("\n\n")
		}
		result.Write(pretty)
		first = false
	}

	if err := scanner.Err(); err != nil {
		return result.String(), err
	}

	return result.String(), nil
}

// processValue recursively processes a value, expanding any JSON strings
func processValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		// Try to parse as JSON
		trimmed := strings.TrimSpace(val)
		if len(trimmed) == 0 {
			return val
		}

		// Only try to parse if it looks like JSON (starts with { or [)
		if (trimmed[0] == '{' || trimmed[0] == '[') {
			var parsed interface{}
			if json.Unmarshal([]byte(val), &parsed) == nil {
				return processValue(parsed) // Recurse on the parsed value
			}
		}
		return val

	case map[string]interface{}:
		for k, v := range val {
			val[k] = processValue(v)
		}
		return val

	case []interface{}:
		for i, v := range val {
			val[i] = processValue(v)
		}
		return val

	default:
		return val
	}
}

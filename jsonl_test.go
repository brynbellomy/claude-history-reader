package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJSONLFile(t *testing.T) {
	// Create a temp file with test data
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"type":"user","content":"{\"strategyName\":\"TestStrategy\"}"}
{"type":"assistant","message":"Hello world"}`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := ParseJSONLFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse JSONL file: %v", err)
	}

	// Check that nested JSON was expanded (not escaped quotes)
	if strings.Contains(content, `\"strategyName\"`) {
		t.Error("Nested JSON was not expanded - still contains escaped quotes")
	}

	// Check that strategyName appears as a proper JSON key
	if !strings.Contains(content, `"strategyName"`) {
		t.Error("strategyName should appear as an unescaped JSON key")
	}

	// Check for 4-space indentation
	if !strings.Contains(content, "    ") {
		t.Error("Output should have 4-space indentation")
	}
}

func TestProcessValue(t *testing.T) {
	// Test nested JSON string expansion
	input := `{"strategyName":"TestStrategy"}`
	result := processValue(input)

	// Should return a map, not a string
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	if m["strategyName"] != "TestStrategy" {
		t.Errorf("Expected strategyName to be TestStrategy, got %v", m["strategyName"])
	}
}

func TestProcessValueNested(t *testing.T) {
	// Test deeply nested JSON
	input := map[string]interface{}{
		"content": `{"inner": "value"}`,
	}
	result := processValue(input)

	m := result.(map[string]interface{})
	inner, ok := m["content"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested content to be a map, got %T", m["content"])
	}

	if inner["inner"] != "value" {
		t.Errorf("Expected inner value to be 'value', got %v", inner["inner"])
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJSONLMessages(t *testing.T) {
	// Create a temp file with test messages
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"type":"user","message":{"role":"user","content":"Hello, how are you?"},"uuid":"test-1","timestamp":"2026-02-02T10:00:00.000Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'm doing well, thank you!"}]},"uuid":"test-2","timestamp":"2026-02-02T10:00:01.000Z"}
{"type":"system","content":"System message here","subtype":"info","uuid":"test-3","timestamp":"2026-02-02T10:00:02.000Z"}
{"type":"summary","summary":"Test conversation","uuid":"test-4","timestamp":"2026-02-02T10:00:03.000Z"}`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	messages, err := ParseJSONLMessages(testFile)
	if err != nil {
		t.Fatalf("ParseJSONLMessages failed: %v", err)
	}

	if len(messages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(messages))
	}

	// Check types
	expectedTypes := []string{"user", "assistant", "system", "summary"}
	for i, msg := range messages {
		if msg.Type != expectedTypes[i] {
			t.Errorf("Message %d: expected type %q, got %q", i, expectedTypes[i], msg.Type)
		}
	}

	// Check user message content
	if len(messages[0].Content) == 0 {
		t.Error("User message should have content")
	} else if messages[0].Content[0].Content != "Hello, how are you?" {
		t.Errorf("User message content mismatch: %q", messages[0].Content[0].Content)
	}

	// Check assistant message content
	if len(messages[1].Content) == 0 {
		t.Error("Assistant message should have content")
	} else if messages[1].Content[0].Type != "text" {
		t.Errorf("Assistant content type should be 'text', got %q", messages[1].Content[0].Type)
	}
}

func TestParseToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Tool result with JSON content that should be prettified
	testData := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"test","content":"{\"strategyName\":\"TestStrategy\"}"}]},"uuid":"test-1","timestamp":"2026-02-02T10:00:00.000Z"}`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	messages, err := ParseJSONLMessages(testFile)
	if err != nil {
		t.Fatalf("ParseJSONLMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if len(messages[0].Content) == 0 {
		t.Fatal("Message should have content")
	}

	block := messages[0].Content[0]
	if block.Type != "tool_result" {
		t.Errorf("Expected type 'tool_result', got %q", block.Type)
	}

	// Content should be prettified JSON
	if block.Content == `{"strategyName":"TestStrategy"}` {
		t.Error("Tool result content should be prettified, not raw")
	}
}

func TestMessageRender(t *testing.T) {
	msg := &Message{
		Type: "user",
		Content: []ContentBlock{
			{Type: "text", Content: "Hello world"},
		},
	}

	rendered := msg.Render(80)
	if rendered == "" {
		t.Error("Rendered message should not be empty")
	}

	// Should contain the type badge
	if !msg.IsImplemented() {
		t.Error("'user' type should be implemented")
	}
}

func TestUnknownTypeWarning(t *testing.T) {
	msg := &Message{
		Type: "unknown_type",
		Content: []ContentBlock{
			{Type: "plain", Content: "Some content"},
		},
	}

	if msg.IsImplemented() {
		t.Error("Unknown type should not be implemented")
	}

	rendered := msg.Render(80)
	// Should contain warning indicator
	if rendered == "" {
		t.Error("Should render something for unknown types")
	}
}

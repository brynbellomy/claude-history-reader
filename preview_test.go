package main

import (
	"testing"
)

func TestExtractStringFromLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantFound bool
	}{
		{
			name:      "simple key-value",
			line:      `    "message": "Hello world",`,
			wantKey:   "message",
			wantValue: "Hello world",
			wantFound: true,
		},
		{
			name:      "key-value without trailing comma",
			line:      `    "content": "Some content"`,
			wantKey:   "content",
			wantValue: "Some content",
			wantFound: true,
		},
		{
			name:      "string with escaped newlines",
			line:      `    "text": "Line 1\nLine 2\nLine 3"`,
			wantKey:   "text",
			wantValue: "Line 1\nLine 2\nLine 3",
			wantFound: true,
		},
		{
			name:      "string with escaped quotes",
			line:      `    "data": "He said \"hello\""`,
			wantKey:   "data",
			wantValue: `He said "hello"`,
			wantFound: true,
		},
		{
			name:      "non-string value (number)",
			line:      `    "count": 42,`,
			wantKey:   "",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "non-string value (boolean)",
			line:      `    "enabled": true`,
			wantKey:   "",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "object start",
			line:      `    "config": {`,
			wantKey:   "",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "array element",
			line:      `        "array item",`,
			wantKey:   "",
			wantValue: "array item",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, found := ExtractStringFromLine(tt.line)
			if found != tt.wantFound {
				t.Errorf("ExtractStringFromLine() found = %v, want %v", found, tt.wantFound)
			}
			if key != tt.wantKey {
				t.Errorf("ExtractStringFromLine() key = %q, want %q", key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("ExtractStringFromLine() value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestRenderPreview(t *testing.T) {
	// Test that markdown rendering works
	line := `    "description": "# Title\n\nThis is **bold** text."`
	preview := RenderPreview(line, 80)

	if !preview.IsString {
		t.Error("Expected IsString to be true")
	}

	if preview.Key != "description" {
		t.Errorf("Expected key 'description', got %q", preview.Key)
	}

	// The rendered output should not be empty
	if len(preview.Rendered) == 0 {
		t.Error("Expected non-empty rendered content")
	}
}

func TestRenderPreviewNoString(t *testing.T) {
	line := `    "count": 42,`
	preview := RenderPreview(line, 80)

	if preview.IsString {
		t.Error("Expected IsString to be false for non-string value")
	}
}

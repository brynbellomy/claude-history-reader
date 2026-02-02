package main

import (
	"strings"
	"testing"
)

func TestHighlightJSON(t *testing.T) {
	input := `{
    "key": "value",
    "number": 42,
    "bool": true,
    "null": null
}`

	result := HighlightJSON(input)

	// Strip any ANSI codes and verify content is preserved
	// Note: lipgloss may not emit ANSI codes in non-TTY test environments
	stripped := StripAnsi(result)
	if stripped != input {
		t.Errorf("Content was modified during highlighting.\nExpected:\n%s\nGot:\n%s", input, stripped)
	}

	// Verify function ran without panicking and produced output
	if len(result) == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestHighlightSearch(t *testing.T) {
	input := "Hello world, hello again"
	result := HighlightSearch(input, "hello")

	// Should contain ANSI escape codes for highlighting
	if !strings.Contains(result, "\x1b[") {
		t.Error("Expected ANSI escape codes for search highlighting")
	}

	// Should highlight both occurrences (case-insensitive)
	// Count the highlight start sequences
	highlightCount := strings.Count(result, "\x1b[1;30;48;5;226m")
	if highlightCount != 2 {
		t.Errorf("Expected 2 highlights, got %d", highlightCount)
	}
}

func TestHighlightSearchEmpty(t *testing.T) {
	input := "Hello world"
	result := HighlightSearch(input, "")

	// Should return unchanged
	if result != input {
		t.Error("Empty query should return unchanged input")
	}
}

func TestHighlightSearchNoMatch(t *testing.T) {
	input := "Hello world"
	result := HighlightSearch(input, "xyz")

	// Should return unchanged (no matches)
	if result != input {
		t.Error("No matches should return unchanged input")
	}
}

func TestStripAnsi(t *testing.T) {
	input := "\x1b[31mRed\x1b[0m text"
	expected := "Red text"
	result := StripAnsi(input)

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

package main

import (
	"testing"
)

func TestPathToClaudeDirName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/Users/brynbellomy/my-project",
			expected: "-Users-brynbellomy-my-project",
		},
		{
			input:    "/Users/brynbellomy/projects/allora/test-robonet-skills",
			expected: "-Users-brynbellomy-projects-allora-test-robonet-skills",
		},
		{
			input:    "/home/user/code",
			expected: "-home-user-code",
		},
		{
			input:    "/path/with spaces/and.dots",
			expected: "-path-with-spaces-and-dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := PathToClaudeDirName(tt.input)
			if result != tt.expected {
				t.Errorf("PathToClaudeDirName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Resolve the appropriate directory for JSONL files
	// If we're not in ~/.claude/projects, look for Claude history for this project
	searchDir, projectPath, err := ResolveJSONLDir(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving directory: %v\n", err)
		os.Exit(1)
	}

	// Find JSONL files
	files, err := FindJSONLFiles(searchDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding JSONL files: %v\n", err)
		os.Exit(1)
	}

	// Create and run the TUI
	model := NewModel(files, projectPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

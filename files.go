package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

type FileInfo struct {
	Path    string
	Name    string
	ModTime time.Time
}

// GetClaudeProjectsDir returns the path to ~/.claude/projects
func GetClaudeProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// PathToClaudeDirName converts an absolute path to Claude's directory naming format
// e.g., /Users/brynbellomy/my-project -> -Users-brynbellomy-my-project
func PathToClaudeDirName(absPath string) string {
	var result strings.Builder
	for _, r := range absPath {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	return result.String()
}

// ResolveJSONLDir determines the appropriate directory to search for JSONL files.
// If cwd is already under ~/.claude/projects, use cwd directly.
// Otherwise, convert cwd to the Claude history path format.
func ResolveJSONLDir(cwd string) (string, string, error) {
	claudeProjectsDir, err := GetClaudeProjectsDir()
	if err != nil {
		return "", "", err
	}

	// Check if cwd is already under ~/.claude/projects
	if strings.HasPrefix(cwd, claudeProjectsDir) {
		return cwd, "", nil
	}

	// Convert cwd to Claude's history directory format
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}

	dirName := PathToClaudeDirName(absPath)
	historyDir := filepath.Join(claudeProjectsDir, dirName)

	// Check if the history directory exists
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		// Return cwd as fallback, but indicate no history was found
		return cwd, "", nil
	}

	return historyDir, absPath, nil
}

// FindJSONLFiles searches the given directory for .jsonl files
// and returns them sorted by modification time (newest first)
func FindJSONLFiles(dir string) ([]FileInfo, error) {
	pattern := filepath.Join(dir, "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Path:    path,
			Name:    filepath.Base(path),
			ModTime: info.ModTime(),
		})
	}

	// Sort by modification time, newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	return files, nil
}

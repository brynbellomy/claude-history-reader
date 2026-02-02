package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// JSON syntax highlighting styles
	keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))  // Cyan for keys
	strStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // Green for strings
	numStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141")) // Purple for numbers
	boolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange for booleans/null
	braceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Gray for braces/brackets

	// Search highlight style
	searchHighlightStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("226")).
		Foreground(lipgloss.Color("0")).
		Bold(true)
)

// HighlightJSON applies syntax highlighting to pretty-printed JSON
func HighlightJSON(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, highlightJSONLine(line))
	}

	return strings.Join(result, "\n")
}

// highlightJSONLine applies syntax highlighting to a single line of JSON
func highlightJSONLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}

	var result strings.Builder
	i := 0
	n := len(line)

	for i < n {
		ch := line[i]

		switch {
		// Handle whitespace (preserve indentation)
		case ch == ' ' || ch == '\t':
			result.WriteByte(ch)
			i++

		// Handle braces and brackets
		case ch == '{' || ch == '}' || ch == '[' || ch == ']':
			result.WriteString(braceStyle.Render(string(ch)))
			i++

		// Handle colons and commas
		case ch == ':' || ch == ',':
			result.WriteByte(ch)
			i++

		// Handle strings (keys or values)
		case ch == '"':
			// Find the end of the string
			j := i + 1
			for j < n && line[j] != '"' {
				if line[j] == '\\' && j+1 < n {
					j += 2 // Skip escaped character
				} else {
					j++
				}
			}
			if j < n {
				j++ // Include closing quote
			}

			str := line[i:j]

			// Check if this is a key (followed by colon)
			restTrimmed := strings.TrimSpace(line[j:])
			if len(restTrimmed) > 0 && restTrimmed[0] == ':' {
				result.WriteString(keyStyle.Render(str))
			} else {
				result.WriteString(strStyle.Render(str))
			}
			i = j

		// Handle numbers
		case ch == '-' || (ch >= '0' && ch <= '9'):
			j := i
			for j < n && (line[j] == '-' || line[j] == '+' || line[j] == '.' ||
				line[j] == 'e' || line[j] == 'E' ||
				(line[j] >= '0' && line[j] <= '9')) {
				j++
			}
			result.WriteString(numStyle.Render(line[i:j]))
			i = j

		// Handle booleans and null
		case ch == 't' || ch == 'f' || ch == 'n':
			// Check for true, false, null
			remaining := line[i:]
			if strings.HasPrefix(remaining, "true") {
				result.WriteString(boolStyle.Render("true"))
				i += 4
			} else if strings.HasPrefix(remaining, "false") {
				result.WriteString(boolStyle.Render("false"))
				i += 5
			} else if strings.HasPrefix(remaining, "null") {
				result.WriteString(boolStyle.Render("null"))
				i += 4
			} else {
				result.WriteByte(ch)
				i++
			}

		default:
			result.WriteByte(ch)
			i++
		}
	}

	return result.String()
}

// HighlightSearch highlights all occurrences of query in the content
// This works on already syntax-highlighted content by operating on visible text
func HighlightSearch(content string, query string) string {
	if query == "" {
		return content
	}

	// Use case-insensitive matching
	queryLower := strings.ToLower(query)
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, highlightSearchInLine(line, queryLower))
	}

	return strings.Join(result, "\n")
}

// highlightSearchInLine highlights search matches in a single line
// Handles ANSI escape codes by tracking visible character positions
func highlightSearchInLine(line string, queryLower string) string {
	if queryLower == "" {
		return line
	}

	// Extract visible text
	visible, _ := extractVisibleText(line)
	visibleLower := strings.ToLower(visible)

	// Find all match positions in visible text
	var matches []int
	pos := 0
	for {
		idx := strings.Index(visibleLower[pos:], queryLower)
		if idx == -1 {
			break
		}
		matches = append(matches, pos+idx)
		pos = pos + idx + 1
	}

	if len(matches) == 0 {
		return line
	}

	// Rebuild the line with highlights
	var result strings.Builder
	lineIdx := 0
	visibleIdx := 0
	matchIdx := 0
	inMatch := false
	matchEnd := 0

	for lineIdx < len(line) {
		// Check if we're at an ANSI escape sequence
		if ansiSeq := getAnsiSequence(line[lineIdx:]); ansiSeq != "" {
			result.WriteString(ansiSeq)
			lineIdx += len(ansiSeq)
			continue
		}

		// Check if we're starting a new match
		if matchIdx < len(matches) && visibleIdx == matches[matchIdx] {
			inMatch = true
			matchEnd = visibleIdx + len(queryLower)
			result.WriteString("\x1b[1;30;48;5;226m") // Bold black on yellow
		}

		// Write the character
		result.WriteByte(line[lineIdx])
		lineIdx++
		visibleIdx++

		// Check if we're ending a match
		if inMatch && visibleIdx == matchEnd {
			result.WriteString("\x1b[0m") // Reset
			// Re-apply any style that was active (simplified: just reset)
			inMatch = false
			matchIdx++
		}
	}

	return result.String()
}

// extractVisibleText extracts text without ANSI codes
func extractVisibleText(line string) (string, map[int]int) {
	var visible strings.Builder
	posMap := make(map[int]int) // visible position -> line position (unused but kept for future)
	lineIdx := 0
	visibleIdx := 0

	for lineIdx < len(line) {
		if ansiSeq := getAnsiSequence(line[lineIdx:]); ansiSeq != "" {
			lineIdx += len(ansiSeq)
			continue
		}
		posMap[visibleIdx] = lineIdx
		visible.WriteByte(line[lineIdx])
		lineIdx++
		visibleIdx++
	}

	_ = posMap // Silence unused warning
	return visible.String(), nil
}

// getAnsiSequence returns the ANSI escape sequence at the start of s, or empty string
func getAnsiSequence(s string) string {
	if len(s) < 2 || s[0] != '\x1b' || s[1] != '[' {
		return ""
	}

	// Find the end of the sequence (letter)
	for i := 2; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			return s[:i+1]
		}
		// Valid sequence characters: digits, semicolon, question mark
		if !((ch >= '0' && ch <= '9') || ch == ';' || ch == '?') {
			break
		}
	}
	return ""
}

// StripAnsi removes ANSI escape codes from a string
func StripAnsi(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

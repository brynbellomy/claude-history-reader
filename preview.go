package main

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	// Style for the preview pane header
	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)
)

// PreviewResult contains the extracted string and its rendered preview
type PreviewResult struct {
	Key      string // The JSON key (if found)
	RawValue string // The raw string value
	Rendered string // Markdown-rendered content
	IsString bool   // Whether a string value was found
}

// ExtractStringFromLine attempts to extract a string value from a JSON line
// Returns the key name and the string value if found
func ExtractStringFromLine(line string) (key string, value string, found bool) {
	trimmed := strings.TrimSpace(line)

	// Try to match a key-value pair where value is a string
	// Pattern: "key": "value" or "key": "value",
	keyValueRegex := regexp.MustCompile(`^"([^"]+)":\s*"((?:[^"\\]|\\.)*)"\s*,?\s*$`)
	if matches := keyValueRegex.FindStringSubmatch(trimmed); matches != nil {
		key = matches[1]
		value = matches[2]
		// Unescape the string
		value = unescapeJSONString(value)
		return key, value, true
	}

	// Try to match just a string value (array element)
	// Pattern: "value" or "value",
	stringOnlyRegex := regexp.MustCompile(`^"((?:[^"\\]|\\.)*)"\s*,?\s*$`)
	if matches := stringOnlyRegex.FindStringSubmatch(trimmed); matches != nil {
		value = matches[1]
		value = unescapeJSONString(value)
		return "", value, true
	}

	return "", "", false
}

// unescapeJSONString unescapes a JSON string value
func unescapeJSONString(s string) string {
	// Wrap in quotes and use json.Unmarshal to properly unescape
	quoted := `"` + s + `"`
	var result string
	if err := json.Unmarshal([]byte(quoted), &result); err != nil {
		// Fallback: manual unescape of common sequences
		result = s
		result = strings.ReplaceAll(result, `\\`, `\`)
		result = strings.ReplaceAll(result, `\"`, `"`)
		result = strings.ReplaceAll(result, `\n`, "\n")
		result = strings.ReplaceAll(result, `\t`, "\t")
		result = strings.ReplaceAll(result, `\r`, "\r")
	}
	return result
}

// RenderPreview extracts a string from the line and renders it as markdown
func RenderPreview(line string, width int) PreviewResult {
	key, value, found := ExtractStringFromLine(line)
	if !found {
		return PreviewResult{IsString: false}
	}

	// Render as markdown
	rendered := renderMarkdown(value, width)

	return PreviewResult{
		Key:      key,
		RawValue: value,
		Rendered: rendered,
		IsString: true,
	}
}

// looksLikeMarkdown checks if content appears to contain markdown formatting
func looksLikeMarkdown(content string) bool {
	// Check for common markdown indicators
	markdownPatterns := []string{
		"```",      // Code blocks
		"**",       // Bold
		"__",       // Bold
		"*",        // Italic (but be careful with bullet points)
		"_",        // Italic
		"# ",       // Headers
		"## ",      // Headers
		"### ",     // Headers
		"- ",       // Lists
		"* ",       // Lists
		"1. ",      // Numbered lists
		"[",        // Links
		"`",        // Inline code
		"> ",       // Blockquotes
		"---",      // Horizontal rules
		"***",      // Horizontal rules
	}

	for _, pattern := range markdownPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// renderMarkdown renders a string as markdown for terminal display
// Falls back to plain text word wrapping for non-markdown content
func renderMarkdown(content string, width int) string {
	if width < 20 {
		width = 20
	}

	// If it doesn't look like markdown, just word wrap it
	if !looksLikeMarkdown(content) {
		return wordwrap.String(content, width)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return wordwrap.String(content, width)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return wordwrap.String(content, width)
	}

	// Trim trailing whitespace
	return strings.TrimRight(rendered, "\n\r\t ")
}

// FormatPreviewPane creates the preview pane content
func FormatPreviewPane(preview PreviewResult, width, height int) string {
	if !preview.IsString {
		msg := noPreviewStyle.Render("No string value on current line")
		return centerVertically(msg, height)
	}

	var b strings.Builder

	// Header showing the key name
	if preview.Key != "" {
		b.WriteString(previewHeaderStyle.Render("Preview: " + preview.Key))
	} else {
		b.WriteString(previewHeaderStyle.Render("Preview"))
	}
	b.WriteString("\n")

	// Rendered content
	b.WriteString(preview.Rendered)

	return b.String()
}

// centerVertically centers text vertically in the given height
func centerVertically(text string, height int) string {
	lines := strings.Split(text, "\n")
	textHeight := len(lines)

	if textHeight >= height {
		return text
	}

	padding := (height - textHeight) / 2
	var b strings.Builder
	for i := 0; i < padding; i++ {
		b.WriteString("\n")
	}
	b.WriteString(text)
	return b.String()
}

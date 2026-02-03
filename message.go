package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// Message represents a parsed JSONL entry
type Message struct {
	Type      string
	Subtype   string // For system messages
	Timestamp time.Time
	UUID      string
	Content   []ContentBlock
	Raw       map[string]interface{}
	IsMeta    bool // Meta messages (like skill loading) can be de-emphasized
}

// ContentBlock represents a piece of content within a message
type ContentBlock struct {
	Type    string // "text", "thinking", "tool_use", "tool_result", "plain"
	Content string // The actual content
	Name    string // For tool_use: tool name
}

// Message type styles
var (
	userBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("27")).
			Padding(0, 1)

	assistantBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("34")).
				Padding(0, 1)

	systemBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("220")).
				Padding(0, 1)

	summaryBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("99")).
				Padding(0, 1)

	unknownBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("240")).
				Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	thinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	toolUseHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("44"))

	toolResultHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214"))

	messageBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	metaMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))
)

// Known/implemented message types
var implementedTypes = map[string]bool{
	"user":      true,
	"assistant": true,
	"system":    true,
	"summary":   true,
}

// ParseJSONLMessages parses a JSONL file into Message structs
func ParseJSONLMessages(path string) ([]Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		msg := parseMessage(raw)
		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	return messages, scanner.Err()
}

func parseMessage(raw map[string]interface{}) *Message {
	msgType, _ := raw["type"].(string)

	// Skip file-history-snapshot as it's not useful to display
	if msgType == "file-history-snapshot" {
		return nil
	}

	msg := &Message{
		Type: msgType,
		Raw:  raw,
	}

	// Parse timestamp
	if ts, ok := raw["timestamp"].(string); ok {
		msg.Timestamp, _ = time.Parse(time.RFC3339, ts)
	}

	// Parse UUID
	msg.UUID, _ = raw["uuid"].(string)

	// Parse isMeta
	msg.IsMeta, _ = raw["isMeta"].(bool)

	// Parse subtype for system messages
	msg.Subtype, _ = raw["subtype"].(string)

	// Extract content based on type
	switch msgType {
	case "user", "assistant":
		msg.Content = parseMessageContent(raw)
	case "system":
		msg.Content = parseSystemContent(raw)
	case "summary":
		msg.Content = parseSummaryContent(raw)
	default:
		// For unknown types, try to extract something useful
		msg.Content = parseUnknownContent(raw)
	}

	return msg
}

func parseMessageContent(raw map[string]interface{}) []ContentBlock {
	var blocks []ContentBlock

	msgObj, ok := raw["message"].(map[string]interface{})
	if !ok {
		return blocks
	}

	content := msgObj["content"]

	// Content can be a string or an array
	switch c := content.(type) {
	case string:
		blocks = append(blocks, ContentBlock{
			Type:    "text",
			Content: c,
		})

	case []interface{}:
		for _, item := range c {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			blockType, _ := itemMap["type"].(string)

			switch blockType {
			case "text":
				text, _ := itemMap["text"].(string)
				blocks = append(blocks, ContentBlock{
					Type:    "text",
					Content: text,
				})

			case "thinking":
				thinking, _ := itemMap["thinking"].(string)
				blocks = append(blocks, ContentBlock{
					Type:    "thinking",
					Content: thinking,
				})

			case "tool_use":
				name, _ := itemMap["name"].(string)
				input := itemMap["input"]
				inputJSON, _ := json.MarshalIndent(input, "", "    ")
				blocks = append(blocks, ContentBlock{
					Type:    "tool_use",
					Name:    name,
					Content: string(inputJSON),
				})

			case "tool_result":
				resultContent, _ := itemMap["content"].(string)
				// Try to parse as JSON and prettify
				var parsed interface{}
				if json.Unmarshal([]byte(resultContent), &parsed) == nil {
					pretty, _ := json.MarshalIndent(parsed, "", "    ")
					resultContent = string(pretty)
				}
				blocks = append(blocks, ContentBlock{
					Type:    "tool_result",
					Content: resultContent,
				})
			}
		}
	}

	return blocks
}

func parseSystemContent(raw map[string]interface{}) []ContentBlock {
	content, _ := raw["content"].(string)
	return []ContentBlock{{
		Type:    "plain",
		Content: content,
	}}
}

func parseSummaryContent(raw map[string]interface{}) []ContentBlock {
	summary, _ := raw["summary"].(string)
	return []ContentBlock{{
		Type:    "plain",
		Content: summary,
	}}
}

func parseUnknownContent(raw map[string]interface{}) []ContentBlock {
	// Try to find any content-like field
	for _, key := range []string{"content", "message", "text", "summary"} {
		if val, ok := raw[key]; ok {
			switch v := val.(type) {
			case string:
				return []ContentBlock{{Type: "plain", Content: v}}
			case map[string]interface{}:
				pretty, _ := json.MarshalIndent(v, "", "    ")
				return []ContentBlock{{Type: "plain", Content: string(pretty)}}
			}
		}
	}

	// Fallback: show raw JSON
	pretty, _ := json.MarshalIndent(raw, "", "    ")
	return []ContentBlock{{Type: "plain", Content: string(pretty)}}
}

// Render renders the message for display
func (m *Message) Render(width int) string {
	var b strings.Builder

	// Badge with type
	badge := m.renderBadge()
	b.WriteString(badge)

	// Warning for unimplemented types
	if !implementedTypes[m.Type] {
		b.WriteString(" ")
		b.WriteString(warningStyle.Render("⚠"))
	}

	b.WriteString("\n")

	// Content
	contentWidth := width - 4 // Account for border padding
	if contentWidth < 20 {
		contentWidth = 20
	}

	content := m.renderContent(contentWidth)

	// Apply meta style if this is a meta message
	if m.IsMeta {
		content = metaMessageStyle.Render(content)
	}

	b.WriteString(content)

	return b.String()
}

func (m *Message) renderBadge() string {
	label := m.Type
	if m.Subtype != "" {
		label = fmt.Sprintf("%s (%s)", m.Type, m.Subtype)
	}

	switch m.Type {
	case "user":
		// Check if this is a tool_result
		for _, block := range m.Content {
			if block.Type == "tool_result" {
				return userBadgeStyle.Render("user") + " " + toolResultHeaderStyle.Render("tool_result")
			}
		}
		return userBadgeStyle.Render(label)
	case "assistant":
		return assistantBadgeStyle.Render(label)
	case "system":
		return systemBadgeStyle.Render(label)
	case "summary":
		return summaryBadgeStyle.Render(label)
	default:
		return unknownBadgeStyle.Render(label)
	}
}

func (m *Message) renderContent(width int) string {
	var parts []string

	for _, block := range m.Content {
		rendered := renderBlock(block, width)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}

	return strings.Join(parts, "\n\n")
}

func renderBlock(block ContentBlock, width int) string {
	switch block.Type {
	case "text":
		// Render as markdown
		return renderMarkdown(block.Content, width)

	case "thinking":
		// Render thinking in dimmed style
		content := renderMarkdown(block.Content, width)
		return thinkingStyle.Render("💭 Thinking:\n" + content)

	case "tool_use":
		// Show tool name and prettified input
		header := toolUseHeaderStyle.Render("🔧 " + block.Name)
		return header + "\n" + block.Content

	case "tool_result":
		// Already prettified in parsing
		header := toolResultHeaderStyle.Render("📤 Result")
		return header + "\n" + block.Content

	case "plain":
		// Plain text with word wrap
		return wordwrap.String(block.Content, width)

	default:
		return wordwrap.String(block.Content, width)
	}
}

// IsImplemented returns whether this message type has proper rendering
func (m *Message) IsImplemented() bool {
	return implementedTypes[m.Type]
}

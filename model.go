package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	StateFileList State = iota
	StateViewer
)

type ViewMode int

const (
	ViewModeJSON ViewMode = iota
	ViewModeMessage
)

type Model struct {
	state       State
	files       []FileInfo
	fileIndex   int
	projectPath string // Original project path (if viewing history for a project)

	// Content - JSON mode
	rawLines         []string // Raw JSON lines (for searching/preview)
	highlightedLines []string // Syntax-highlighted lines

	// Content - Message mode
	messages     []Message // Parsed messages
	messageIndex int       // Current message (0-indexed)

	// View mode
	viewMode ViewMode

	// Viewer state (JSON mode)
	cursorLine   int // Current line (0-indexed)
	scrollOffset int // First visible line

	// Viewer state (Message mode)
	messageScrollOffset int // Scroll offset within rendered messages

	// Shared state
	searchQuery string
	searchInput string
	searchMode  bool
	numBuffer   string // For vim number prefix (e.g., "10" in "10j")
	lastKey     string // Track last key for "gg" detection

	// Dimensions
	width  int
	height int
	ready  bool
	err    error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	cursorLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(6).
			Align(lipgloss.Right)

	lineNumberSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Width(6).
				Align(lipgloss.Right).
				Bold(true)
)

func NewModel(files []FileInfo, projectPath string) Model {
	return Model{
		state:       StateFileList,
		files:       files,
		fileIndex:   0,
		projectPath: projectPath,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode input
		if m.searchMode {
			return m.handleSearchInput(msg)
		}

		// Handle based on state
		switch m.state {
		case StateFileList:
			return m.handleFileListKeys(msg)
		case StateViewer:
			return m.handleViewerKeys(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	return m, nil
}

func (m Model) handleFileListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.fileIndex < len(m.files)-1 {
			m.fileIndex++
		}

	case "k", "up":
		if m.fileIndex > 0 {
			m.fileIndex--
		}

	case "enter":
		if len(m.files) > 0 {
			filePath := m.files[m.fileIndex].Path

			// Parse for JSON mode
			content, err := ParseJSONLFile(filePath)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.rawLines = strings.Split(content, "\n")
			highlighted := HighlightJSON(content)
			m.highlightedLines = strings.Split(highlighted, "\n")

			// Parse for Message mode
			messages, err := ParseJSONLMessages(filePath)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.messages = messages

			// Reset state
			m.cursorLine = 0
			m.scrollOffset = 0
			m.messageIndex = 0
			m.messageScrollOffset = 0
			m.viewMode = ViewModeMessage // Start in message mode
			m.state = StateViewer
			m.searchQuery = ""
		}

	case "g":
		if m.lastKey == "g" {
			m.fileIndex = 0
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
		return m, nil

	case "G":
		m.fileIndex = len(m.files) - 1
	}

	if msg.String() != "g" {
		m.lastKey = ""
	}

	return m, nil
}

func (m Model) handleViewerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Tab toggles view mode
	if key == "tab" {
		if m.viewMode == ViewModeJSON {
			m.viewMode = ViewModeMessage
		} else {
			m.viewMode = ViewModeJSON
		}
		return m, nil
	}

	// Handle number prefix for vim commands
	if key >= "0" && key <= "9" {
		if key == "0" && m.numBuffer == "" {
			// "0" by itself - go to top
			if m.viewMode == ViewModeJSON {
				m.cursorLine = 0
				m.ensureCursorVisible()
			} else {
				m.messageIndex = 0
				m.messageScrollOffset = 0
			}
			return m, nil
		}
		m.numBuffer += key
		return m, nil
	}

	count := 1
	if m.numBuffer != "" {
		n, err := strconv.Atoi(m.numBuffer)
		if err == nil && n > 0 {
			count = n
		}
		m.numBuffer = ""
	}

	switch key {
	case "q":
		m.state = StateFileList
		m.searchQuery = ""
		m.searchInput = ""
		return m, nil

	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.searchInput = ""
			return m, nil
		}
		m.state = StateFileList
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.viewMode == ViewModeJSON {
			m.handleJSONNavigation(count, 1)
		} else {
			m.handleMessageNavigation(count, 1)
		}

	case "k", "up":
		if m.viewMode == ViewModeJSON {
			m.handleJSONNavigation(count, -1)
		} else {
			m.handleMessageNavigation(count, -1)
		}

	case "ctrl+d":
		viewHeight := m.viewerHeight()
		if m.viewMode == ViewModeJSON {
			m.handleJSONNavigation(viewHeight/2, 1)
		} else {
			m.handleMessageNavigation(viewHeight/2, 1)
		}

	case "ctrl+u":
		viewHeight := m.viewerHeight()
		if m.viewMode == ViewModeJSON {
			m.handleJSONNavigation(viewHeight/2, -1)
		} else {
			m.handleMessageNavigation(viewHeight/2, -1)
		}

	case "g":
		if m.lastKey == "g" {
			if m.viewMode == ViewModeJSON {
				m.cursorLine = 0
				m.scrollOffset = 0
			} else {
				m.messageIndex = 0
				m.messageScrollOffset = 0
			}
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
		return m, nil

	case "G":
		if m.viewMode == ViewModeJSON {
			m.cursorLine = len(m.rawLines) - 1
			m.ensureCursorVisible()
		} else {
			m.messageIndex = len(m.messages) - 1
			m.messageScrollOffset = 0
		}

	case "/":
		m.searchMode = true
		m.searchInput = ""
		return m, nil

	case "n":
		if m.searchQuery != "" {
			m.findNext(1)
		}

	case "N":
		if m.searchQuery != "" {
			m.findNext(-1)
		}
	}

	if key != "g" {
		m.lastKey = ""
	}

	return m, nil
}

func (m *Model) handleJSONNavigation(count, direction int) {
	totalLines := len(m.rawLines)
	m.cursorLine += count * direction
	if m.cursorLine >= totalLines {
		m.cursorLine = totalLines - 1
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	m.ensureCursorVisible()
}

func (m *Model) handleMessageNavigation(count, direction int) {
	totalMessages := len(m.messages)
	if totalMessages == 0 {
		return
	}
	m.messageIndex += count * direction
	if m.messageIndex >= totalMessages {
		m.messageIndex = totalMessages - 1
	}
	if m.messageIndex < 0 {
		m.messageIndex = 0
	}
	m.messageScrollOffset = 0 // Reset scroll when changing messages
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput
		m.searchMode = false
		if m.searchQuery != "" {
			m.findNext(1)
		}
		return m, nil

	case "esc":
		m.searchMode = false
		m.searchInput = ""
		return m, nil

	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.searchInput += msg.String()
		}
		return m, nil
	}
}

// viewerHeight returns the number of visible lines in the viewer
func (m Model) viewerHeight() int {
	return m.height - 5 // header + divider + footer + padding
}

// leftPaneWidth returns the width of the JSON pane (left side)
func (m Model) leftPaneWidth() int {
	return m.width * 55 / 100 // 55% for JSON
}

// rightPaneWidth returns the width of the preview pane (right side)
func (m Model) rightPaneWidth() int {
	return m.width - m.leftPaneWidth() - 3 // 3 for separator
}

// ensureCursorVisible adjusts scroll to keep cursor in view
func (m *Model) ensureCursorVisible() {
	viewHeight := m.viewerHeight()
	if viewHeight <= 0 {
		viewHeight = 10
	}

	// If cursor is above visible area, scroll up
	if m.cursorLine < m.scrollOffset {
		m.scrollOffset = m.cursorLine
	}

	// If cursor is below visible area, scroll down
	if m.cursorLine >= m.scrollOffset+viewHeight {
		m.scrollOffset = m.cursorLine - viewHeight + 1
	}
}

func (m *Model) findNext(direction int) {
	if m.searchQuery == "" {
		return
	}

	totalLines := len(m.rawLines)
	queryLower := strings.ToLower(m.searchQuery)

	for i := 1; i <= totalLines; i++ {
		var lineIdx int
		if direction > 0 {
			lineIdx = (m.cursorLine + i) % totalLines
		} else {
			lineIdx = (m.cursorLine - i + totalLines) % totalLines
		}

		if strings.Contains(strings.ToLower(m.rawLines[lineIdx]), queryLower) {
			m.cursorLine = lineIdx
			m.ensureCursorVisible()
			return
		}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	switch m.state {
	case StateFileList:
		return m.viewFileList()
	case StateViewer:
		if m.viewMode == ViewModeMessage {
			return m.viewMessageMode()
		}
		return m.viewJSONMode()
	default:
		return ""
	}
}

func (m Model) viewFileList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Claude JSONL Viewer"))
	b.WriteString("\n")

	if m.projectPath != "" {
		b.WriteString(helpStyle.Render("Project: " + m.projectPath))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.files) == 0 {
		if m.projectPath != "" {
			b.WriteString("No Claude history found for this project.\n")
		} else {
			b.WriteString("No .jsonl files found in current directory.\n")
		}
	} else {
		for i, f := range m.files {
			line := fmt.Sprintf("  %s  (%s)", f.Name, f.ModTime.Format("2006-01-02 15:04:05"))
			if i == m.fileIndex {
				b.WriteString(selectedStyle.Render("> "+line) + "\n")
			} else {
				b.WriteString(normalStyle.Render("  "+line) + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate • enter: open • q: quit"))

	return b.String()
}

func (m Model) viewJSONMode() string {
	var b strings.Builder

	// Header
	fileName := ""
	if m.fileIndex < len(m.files) {
		fileName = m.files[m.fileIndex].Name
	}
	header := titleStyle.Render(fileName)
	if m.searchQuery != "" {
		header += "  " + searchStyle.Render(fmt.Sprintf("[/%s]", m.searchQuery))
	}
	lineInfo := helpStyle.Render(fmt.Sprintf("Line %d/%d", m.cursorLine+1, len(m.rawLines)))
	headerPadding := m.width - lipgloss.Width(header) - lipgloss.Width(lineInfo)
	if headerPadding < 1 {
		headerPadding = 1
	}
	b.WriteString(header + strings.Repeat(" ", headerPadding) + lineInfo)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Two-column layout
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()
	viewHeight := m.viewerHeight()

	// Build left pane (JSON with cursor)
	leftLines := m.buildLeftPane(leftWidth, viewHeight)

	// Build right pane (preview)
	rightLines := m.buildRightPane(rightWidth, viewHeight)

	// Combine columns
	for i := 0; i < viewHeight; i++ {
		leftLine := ""
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		rightLine := ""
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		// Pad left line to width
		leftLine = padOrTruncate(leftLine, leftWidth)

		b.WriteString(leftLine)
		b.WriteString(" │ ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Footer
	if m.searchMode {
		b.WriteString(searchStyle.Render(fmt.Sprintf("/%s", m.searchInput)))
	} else {
		progress := ""
		if len(m.rawLines) > 0 {
			pct := (m.cursorLine + 1) * 100 / len(m.rawLines)
			progress = fmt.Sprintf("%d%%", pct)
		}
		help := helpStyle.Render("j/k: move • Tab: message mode • /: search • q: back")
		b.WriteString(fmt.Sprintf("%s  %s", progress, help))
	}

	return b.String()
}

// buildLeftPane builds the JSON viewer with cursor highlighting
func (m Model) buildLeftPane(width, height int) []string {
	var lines []string
	lineNumWidth := 6

	for i := 0; i < height; i++ {
		lineIdx := m.scrollOffset + i
		if lineIdx >= len(m.highlightedLines) {
			lines = append(lines, "")
			continue
		}

		// Line number
		var lineNum string
		if lineIdx == m.cursorLine {
			lineNum = lineNumberSelectedStyle.Render(fmt.Sprintf("%d", lineIdx+1))
		} else {
			lineNum = lineNumberStyle.Render(fmt.Sprintf("%d", lineIdx+1))
		}

		// Content (use highlighted version, apply search highlighting if needed)
		content := m.highlightedLines[lineIdx]
		if m.searchQuery != "" {
			content = HighlightSearch(content, m.searchQuery)
		}

		// Truncate content to fit
		maxContentWidth := width - lineNumWidth - 2
		content = truncateWithAnsi(content, maxContentWidth)

		// Apply cursor line background
		if lineIdx == m.cursorLine {
			// Pad content and apply background
			content = padOrTruncate(content, maxContentWidth)
			content = cursorLineStyle.Render(content)
		}

		lines = append(lines, lineNum+" "+content)
	}

	return lines
}

// buildRightPane builds the preview pane
func (m Model) buildRightPane(width, height int) []string {
	if m.cursorLine >= len(m.rawLines) {
		return []string{noPreviewStyle.Render("No content")}
	}

	// Get the current line's raw content
	rawLine := m.rawLines[m.cursorLine]

	// Try to extract and render a string value
	preview := RenderPreview(rawLine, width-2) // -2 for padding

	if !preview.IsString {
		msg := noPreviewStyle.Render("(no string value)")
		return centerLines([]string{msg}, height)
	}

	// Build preview content
	var content strings.Builder

	// Header
	if preview.Key != "" {
		content.WriteString(titleStyle.Render(preview.Key))
	} else {
		content.WriteString(titleStyle.Render("String Value"))
	}
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", width-2))
	content.WriteString("\n")

	// Rendered content (already word-wrapped by glamour)
	content.WriteString(preview.Rendered)

	// Split into lines - glamour already handles word wrapping
	lines := strings.Split(content.String(), "\n")

	return lines
}

var noPreviewStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Italic(true)

// padOrTruncate ensures a string (with possible ANSI codes) fits exactly in width
func padOrTruncate(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible > width {
		return truncateWithAnsi(s, width)
	}
	if visible < width {
		return s + strings.Repeat(" ", width-visible)
	}
	return s
}

// truncateWithAnsi truncates a string with ANSI codes to a visible width
func truncateWithAnsi(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var result strings.Builder
	visibleWidth := 0
	i := 0

	for i < len(s) && visibleWidth < maxWidth {
		// Check for ANSI escape sequence
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Find end of sequence
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++ // Include the final letter
			}
			result.WriteString(s[i:j])
			i = j
			continue
		}

		// Regular character
		result.WriteByte(s[i])
		visibleWidth++
		i++
	}

	// Reset ANSI at end to be safe
	result.WriteString("\x1b[0m")
	return result.String()
}

// centerLines centers lines vertically
func centerLines(lines []string, height int) []string {
	if len(lines) >= height {
		return lines[:height]
	}

	padding := (height - len(lines)) / 2
	result := make([]string, height)
	for i := 0; i < padding; i++ {
		result[i] = ""
	}
	for i, line := range lines {
		result[padding+i] = line
	}
	return result
}

// viewMessageMode renders the message-focused view
func (m Model) viewMessageMode() string {
	var b strings.Builder

	// Header
	fileName := ""
	if m.fileIndex < len(m.files) {
		fileName = m.files[m.fileIndex].Name
	}
	header := titleStyle.Render(fileName)

	// Message counter
	msgInfo := ""
	if len(m.messages) > 0 {
		msgInfo = helpStyle.Render(fmt.Sprintf("Message %d/%d", m.messageIndex+1, len(m.messages)))
	}

	// Mode indicator
	modeIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render("[MSG]")

	headerPadding := m.width - lipgloss.Width(header) - lipgloss.Width(msgInfo) - lipgloss.Width(modeIndicator) - 2
	if headerPadding < 1 {
		headerPadding = 1
	}
	b.WriteString(header + strings.Repeat(" ", headerPadding) + msgInfo + " " + modeIndicator)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Content area
	viewHeight := m.viewerHeight()
	contentWidth := m.width - 2

	if len(m.messages) == 0 {
		b.WriteString(helpStyle.Render("No messages to display"))
	} else {
		// Render current message
		msg := m.messages[m.messageIndex]
		rendered := msg.Render(contentWidth)
		lines := strings.Split(rendered, "\n")

		// Apply scroll offset and show visible lines
		startLine := m.messageScrollOffset
		if startLine >= len(lines) {
			startLine = 0
		}

		for i := 0; i < viewHeight && startLine+i < len(lines); i++ {
			b.WriteString(lines[startLine+i])
			b.WriteString("\n")
		}

		// Fill remaining height
		visibleLines := len(lines) - startLine
		if visibleLines < viewHeight {
			for i := visibleLines; i < viewHeight; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer
	if m.searchMode {
		b.WriteString(searchStyle.Render(fmt.Sprintf("/%s", m.searchInput)))
	} else {
		help := helpStyle.Render("j/k: prev/next msg • Tab: JSON mode • gg/G: first/last • q: back")
		b.WriteString(help)
	}

	return b.String()
}

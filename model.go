package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	StateFileList State = iota
	StateViewer
	StateSearch
)

type Model struct {
	state              State
	files              []FileInfo
	fileIndex          int
	rawContent         string // Plain JSON content (for searching)
	highlightedContent string // Syntax-highlighted content
	viewport           viewport.Model
	searchQuery        string
	searchInput        string
	searchMode         bool
	numBuffer          string // For vim number prefix (e.g., "10" in "10j")
	width              int
	height             int
	ready              bool
	err                error
	lastKey            string // Track last key for "gg" detection
	projectPath        string // Original project path (if viewing history for a project)
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
	var cmd tea.Cmd
	var cmds []tea.Cmd

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

		headerHeight := 3
		footerHeight := 2
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}
	}

	if m.state == StateViewer {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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
			content, err := ParseJSONLFile(m.files[m.fileIndex].Path)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.rawContent = content
			m.highlightedContent = HighlightJSON(content)
			m.viewport.SetContent(m.highlightedContent)
			m.viewport.GotoTop()
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

	// Handle number prefix for vim commands
	if key >= "0" && key <= "9" {
		// Don't allow leading zeros (except for "0" which could be a command)
		if key == "0" && m.numBuffer == "" {
			// "0" by itself - go to beginning of line (we'll treat as top)
			m.viewport.GotoTop()
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
			// Clear search but stay in viewer
			m.searchQuery = ""
			m.searchInput = ""
			m.updateViewportContent()
			return m, nil
		}
		m.state = StateFileList
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		m.viewport.ScrollDown(count)

	case "k", "up":
		m.viewport.ScrollUp(count)

	case "ctrl+d":
		m.viewport.HalfPageDown()

	case "ctrl+u":
		m.viewport.HalfPageUp()

	case "g":
		if m.lastKey == "g" {
			m.viewport.GotoTop()
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
		return m, nil

	case "G":
		m.viewport.GotoBottom()

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

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput
		m.searchMode = false
		m.updateViewportContent()
		if m.searchQuery != "" {
			m.findNext(1)
		}
		return m, nil

	case "esc":
		m.searchMode = false
		m.searchInput = ""
		m.searchQuery = ""
		m.updateViewportContent()
		return m, nil

	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		return m, nil

	default:
		// Only add printable characters
		if len(msg.String()) == 1 {
			m.searchInput += msg.String()
		}
		return m, nil
	}
}

// updateViewportContent refreshes the viewport with syntax + search highlighting
func (m *Model) updateViewportContent() {
	content := m.highlightedContent
	if m.searchQuery != "" {
		content = HighlightSearch(content, m.searchQuery)
	}
	m.viewport.SetContent(content)
}

func (m *Model) findNext(direction int) {
	if m.searchQuery == "" {
		return
	}

	lines := strings.Split(m.rawContent, "\n")
	currentLine := m.viewport.YOffset
	totalLines := len(lines)

	// Search from current position
	for i := 1; i <= totalLines; i++ {
		var lineIdx int
		if direction > 0 {
			lineIdx = (currentLine + i) % totalLines
		} else {
			lineIdx = (currentLine - i + totalLines) % totalLines
		}

		if strings.Contains(strings.ToLower(lines[lineIdx]), strings.ToLower(m.searchQuery)) {
			m.viewport.SetYOffset(lineIdx)
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
		return m.viewViewer()
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

func (m Model) viewViewer() string {
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
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Content
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Footer
	if m.searchMode {
		b.WriteString(searchStyle.Render(fmt.Sprintf("/%s", m.searchInput)))
	} else {
		progress := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
		help := helpStyle.Render("j/k: scroll • /: search • n/N: next/prev • q: back")
		b.WriteString(fmt.Sprintf("%s  %s", progress, help))
	}

	return b.String()
}

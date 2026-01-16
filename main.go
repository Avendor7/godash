package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Panel int

const (
	PanelTUI Panel = iota
	PanelDirs
	PanelShortcuts
)

type Item struct {
	Title       string
	Description string
	Path        string
}

type Model struct {
	focus  Panel
	width  int
	height int

	topItems    []Item
	midItems    []Item
	bottomItems []Item

	topCursor    int
	midCursor    int
	bottomCursor int

	rightTitle string
	rightBody  string
}

func initialModel() *Model {
	return &Model{
		focus: PanelTUI,
		topItems: []Item{
			{Title: "TUI Apps: Clock", Description: "A terminal clock widget"},
			{Title: "TUI Apps: Calendar", Description: "Calendar in terminal"},
			{Title: "TUI Apps: Notes", Description: "Notes app in terminal"},
			{Title: "TUI Apps: Todo", Description: "Tiny todo app"},
		},
		midItems: []Item{
			{Title: "Dev Projects", Path: "/home/you/projects"},
			{Title: "Work", Path: "/home/you/work"},
			{Title: "Docs", Path: "/home/you/docs"},
		},
		bottomItems: []Item{
			{Title: "Home", Path: "/home/you"},
			{Title: "Downloads", Path: "/home/you/Downloads"},
			{Title: "Configs", Path: "/home/you/.config"},
		},
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) cycleFocus() {
	m.focus = (m.focus + 1) % 3
}

func (m *Model) navigateUp() {
	switch m.focus {
	case PanelTUI:
		if m.topCursor > 0 {
			m.topCursor--
		}
	case PanelDirs:
		if m.midCursor > 0 {
			m.midCursor--
		}
	case PanelShortcuts:
		if m.bottomCursor > 0 {
			m.bottomCursor--
		}
	}
}

func (m *Model) navigateDown() {
	switch m.focus {
	case PanelTUI:
		if m.topCursor < len(m.topItems)-1 {
			m.topCursor++
		}
	case PanelDirs:
		if m.midCursor < len(m.midItems)-1 {
			m.midCursor++
		}
	case PanelShortcuts:
		if m.bottomCursor < len(m.bottomItems)-1 {
			m.bottomCursor++
		}
	}
}

func (m *Model) halfName(title string) string {
	parts := strings.SplitN(title, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(title)
}

func (m *Model) renderHTOPPreview(name string) string {
	// Simple, static HTOP-like preview for demonstration
	return fmt.Sprintf("Preview: %s\nCPU  12%%  [||        ]\nMem  47%%  [||||      ]\nTasks 42 (threads: 6)\nUp 1d 2h", name)
}

func (m *Model) activateCurrent() {
	switch m.focus {
	case PanelTUI:
		if m.topCursor < len(m.topItems) {
			it := m.topItems[m.topCursor]
			name := m.halfName(it.Title)
			m.rightTitle = "HTOP Preview: " + name
			m.rightBody = m.renderHTOPPreview(name)
		}
	case PanelDirs:
		if m.midCursor < len(m.midItems) {
			it := m.midItems[m.midCursor]
			name := m.halfName(it.Title)
			m.rightTitle = "HTOP Preview: " + name
			m.rightBody = m.renderHTOPPreview(name)
		}
	case PanelShortcuts:
		if m.bottomCursor < len(m.bottomItems) {
			it := m.bottomItems[m.bottomCursor]
			name := m.halfName(it.Title)
			m.rightTitle = "HTOP Preview: " + name
			m.rightBody = m.renderHTOPPreview(name)
		}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.cycleFocus()
		case "up", "k":
			m.navigateUp()
		case "down", "j":
			m.navigateDown()
		case "enter", " ":
			m.activateCurrent()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		m.width = 120
	}
	if m.height == 0 {
		m.height = 30
	}
	leftW := 28
	rightW := m.width - leftW - 3
	if rightW < 20 {
		rightW = 20
	}

	leftLines := renderPanelBlock("TUI Apps", m.topItems, m.topCursor, m.focus == PanelTUI, leftW)
	leftLines = append(leftLines, renderPanelBlock("Common Directories", m.midItems, m.midCursor, m.focus == PanelDirs, leftW)...)
	leftLines = append(leftLines, renderPanelBlock("Shortcuts", m.bottomItems, m.bottomCursor, m.focus == PanelShortcuts, leftW)...)

	rightLines := renderRightPane(m.rightTitle, m.rightBody, rightW)

	max := len(leftLines)
	if len(rightLines) > max {
		max = len(rightLines)
	}

	var s []string
	for i := 0; i < max; i++ {
		var ll string
		if i < len(leftLines) {
			ll = leftLines[i]
		} else {
			ll = strings.Repeat(" ", leftW)
		}
		var rr string
		if i < len(rightLines) {
			rr = rightLines[i]
		} else {
			rr = ""
		}
		rr = padRight(rr, rightW)
		s = append(s, fmt.Sprintf("│%s│%s│", ll, rr))
	}

	top := fmt.Sprintf("┌%s┬%s┐", strings.Repeat("─", leftW), strings.Repeat("─", rightW))
	bottom := fmt.Sprintf("└%s┴%s┘", strings.Repeat("─", leftW), strings.Repeat("─", rightW))
	return top + "\n" + strings.Join(s, "\n") + "\n" + bottom
}

func renderPanelBlock(title string, items []Item, cursor int, focused bool, width int) []string {
	lines := []string{padCenter(title, width)}
	for i, it := range items {
		prefix := "  "
		if focused && i == cursor {
			prefix = "> "
		}
		text := prefix + it.Title
		if len(text) > width {
			text = text[:width]
		}
		line := padRight(text, width)
		if focused && i == cursor {
			line = fmt.Sprintf("\x1b[7m%s\x1b[0m", line)
		}
		lines = append(lines, line)
	}
	for len(lines) < 4 {
		lines = append(lines, repeatSpace(width))
	}
	lines = append(lines, repeatChar("─", width))
	return lines
}

func renderRightPane(title, body string, width int) []string {
	lines := []string{padCenter("Details", width)}
	lines = append(lines, repeatSpace(width))
	if title != "" {
		lines = append(lines, padRight("Active: "+title, width))
		lines = append(lines, repeatSpace(width))
	} else {
		lines = append(lines, padRight("Active: None", width))
		lines = append(lines, repeatSpace(width))
	}
	if body != "" {
		for _, line := range strings.Split(body, "\n") {
			lines = append(lines, padRight(line, width))
		}
	}
	for len(lines) < 6 {
		lines = append(lines, repeatSpace(width))
	}
	return lines
}

func padCenter(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	pad := width - len(text)
	left := pad / 2
	right := pad - left
	return repeatSpace(left) + text + repeatSpace(right)
}

func padRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + repeatSpace(width-len(text))
}

func repeatSpace(n int) string          { return strings.Repeat(" ", n) }
func repeatChar(c string, n int) string { return strings.Repeat(c, n) }

func main() {
	program := initialModel()
	p := tea.NewProgram(program)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Panel int

const (
	NavigationPanel Panel = iota
	MainPanel
	StatusPanel
)

type model struct {
	panel      Panel            // current active panel
	navItems   []string         // navigation items in left panel
	navCursor  int              // cursor in navigation panel
	choices    []string         // items in main panel (grocery list)
	mainCursor int              // cursor in main panel
	selected   map[int]struct{} // which to-do items are selected
	width      int              // terminal width
	height     int              // terminal height
}

func initialModel() model {
	return model{
		panel:    MainPanel,
		navItems: []string{"Groceries", "Tasks", "Notes", "Settings"},
		choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi", "Buy cheese", "Buy burgers"},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// Switch between panels
		case "tab":
			switch m.panel {
			case NavigationPanel:
				m.panel = MainPanel
				// Ensure main cursor is in bounds
				if m.mainCursor >= len(m.choices) && len(m.choices) > 0 {
					m.mainCursor = len(m.choices) - 1
				}
			case MainPanel:
				m.panel = NavigationPanel
				// Ensure nav cursor is in bounds
				if m.navCursor >= len(m.navItems) && len(m.navItems) > 0 {
					m.navCursor = len(m.navItems) - 1
				}
			}

		// Navigation controls based on active panel
		case "up", "k":
			switch m.panel {
			case NavigationPanel:
				if m.navCursor > 0 {
					m.navCursor--
				}
			case MainPanel:
				if m.mainCursor > 0 {
					m.mainCursor--
				}
			}

		case "down", "j":
			switch m.panel {
			case NavigationPanel:
				if m.navCursor < len(m.navItems)-1 {
					m.navCursor++
				}
			case MainPanel:
				if m.mainCursor < len(m.choices)-1 {
					m.mainCursor++
				}
			}

		// Toggle selection in main panel
		case "enter", " ":
			if m.panel == MainPanel {
				_, ok := m.selected[m.mainCursor]
				if ok {
					delete(m.selected, m.mainCursor)
				} else {
					m.selected[m.mainCursor] = struct{}{}
				}
			}
		}

	// Handle window resize
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		m.width = 80
		m.height = 24
	}

	// Panel dimensions
	navWidth := 25
	mainWidth := m.width - navWidth - 3 // -3 for borders and spacing
	mainHeight := m.height - 6          // -6 for headers, borders, and status bar

	// Build the UI
	s := ""

	// Top border
	s += fmt.Sprintf("┌%s┬%s┐\n", repeat("─", navWidth), repeat("─", mainWidth))

	// Navigation panel header
	s += fmt.Sprintf("│%s│%s│\n",
		padCenter("Navigation", navWidth),
		padCenter("Grocery List", mainWidth))

	// Separator
	s += fmt.Sprintf("├%s┼%s┤\n", repeat("─", navWidth), repeat("─", mainWidth))

	// Content rows
	for i := 0; i < mainHeight; i++ {
		// Navigation content
		navContent := repeat(" ", navWidth)
		if i < len(m.navItems) {
			cursor := " "
			if m.panel == NavigationPanel && m.navCursor == i {
				cursor = ">"
			}
			item := m.navItems[i]
			navContent = fmt.Sprintf("%s %s", cursor, item)
			if len(navContent) > navWidth {
				navContent = navContent[:navWidth]
			}
			navContent = padRight(navContent, navWidth)

			// Add highlighting for active panel cursor
			if m.panel == NavigationPanel && m.navCursor == i {
				navContent = fmt.Sprintf("\x1b[7m%s\x1b[0m", navContent)
			}
		}

		// Main content
		mainContent := repeat(" ", mainWidth)
		if i < len(m.choices) {
			cursor := " "
			if m.panel == MainPanel && m.mainCursor == i {
				cursor = ">"
			}
			checked := " "
			if _, ok := m.selected[i]; ok {
				checked = "x"
			}
			item := fmt.Sprintf("%s [%s] %s", cursor, checked, m.choices[i])
			if len(item) > mainWidth {
				item = item[:mainWidth]
			}
			mainContent = padRight(item, mainWidth)

			// Add highlighting for active panel cursor
			if m.panel == MainPanel && m.mainCursor == i {
				mainContent = fmt.Sprintf("\x1b[7m%s\x1b[0m", mainContent)
			}
		}

		s += fmt.Sprintf("│%s│%s│\n", navContent, mainContent)
	}

	// Bottom separator before status
	s += fmt.Sprintf("├%s┼%s┤\n", repeat("─", navWidth), repeat("─", mainWidth))

	// Status bar
	statusText := "Tab: Switch panels | ↑↓: Navigate | Space: Toggle | q: Quit"
	panelText := fmt.Sprintf("Active: %s",
		map[Panel]string{NavigationPanel: "Navigation", MainPanel: "Main"}[m.panel])

	s += fmt.Sprintf("│%s│%s│\n",
		padRight(panelText, navWidth),
		padRight(statusText, mainWidth))

	// Bottom border
	s += fmt.Sprintf("└%s┴%s┘", repeat("─", navWidth), repeat("─", mainWidth))

	return s
}

// Helper functions
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func padCenter(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := width - len(text)
	left := padding / 2
	right := padding - left
	return repeat(" ", left) + text + repeat(" ", right)
}

func padRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + repeat(" ", width-len(text))
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

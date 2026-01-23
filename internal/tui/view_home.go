package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agbru/fibcalc/internal/tui/views"
)

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	menuItems := views.HomeMenuItems()
	numItems := len(menuItems)

	switch {
	case key.Matches(msg, m.keys.Up):
		m.homeState.cursor--
		if m.homeState.cursor < 0 {
			m.homeState.cursor = numItems - 1
		}

	case key.Matches(msg, m.keys.Down):
		m.homeState.cursor++
		if m.homeState.cursor >= numItems {
			m.homeState.cursor = 0
		}

	case key.Matches(msg, m.keys.Enter):
		return m.selectHomeMenuItem(m.homeState.cursor)

	case key.Matches(msg, m.keys.NewCalc):
		m.currentView = ViewCalculator
		return m, nil

	case key.Matches(msg, m.keys.Compare):
		return m.startComparison()

	case key.Matches(msg, m.keys.Settings):
		m.currentView = ViewSettings
		return m, nil
	}

	return m, nil
}

func (m Model) selectHomeMenuItem(index int) (tea.Model, tea.Cmd) {
	menuItems := views.HomeMenuItems()
	if index < 0 || index >= len(menuItems) {
		return m, nil
	}

	item := menuItems[index]
	switch item.Key {
	case "c":
		m.currentView = ViewCalculator
	case "m":
		return m.startComparison()
	case "s":
		m.currentView = ViewSettings
	case "?":
		m.prevView = m.currentView
		m.currentView = ViewHelp
	case "q":
		m.cancel()
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) viewHome() string {
	menuItems := views.HomeMenuItems()

	var b strings.Builder

	// Welcome message
	b.WriteString(m.styles.Title.Render("Welcome to Fibonacci Calculator TUI"))
	b.WriteString("\n")
	b.WriteString(m.styles.Subtitle.Render("High-performance Fibonacci number calculation"))
	b.WriteString("\n\n")

	// Current configuration
	configBox := m.styles.Box.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.BoxTitle.Render("Current Configuration"),
			"",
			fmt.Sprintf("%s N: %s%d%s",
				m.styles.ResultLabel.Render(""),
				m.styles.Highlight.Render(""),
				m.config.N,
				m.styles.Muted.Render("")),
			fmt.Sprintf("%s Algorithm: %s%s",
				m.styles.ResultLabel.Render(""),
				m.styles.Highlight.Render(""),
				m.config.Algo),
			fmt.Sprintf("%s Timeout: %s%s",
				m.styles.ResultLabel.Render(""),
				m.styles.Highlight.Render(""),
				m.config.Timeout.String()),
		),
	)
	b.WriteString(configBox)
	b.WriteString("\n\n")

	// Menu
	b.WriteString(m.styles.Bold.Render("Menu"))
	b.WriteString("\n\n")

	for i, item := range menuItems {
		cursor := "  "
		style := m.styles.MenuItem
		if i == m.homeState.cursor {
			cursor = m.styles.Primary.Render("> ")
			style = m.styles.MenuItemActive
		}

		line := fmt.Sprintf("%s[%s] %s",
			cursor,
			m.styles.HelpKey.Render(item.Key),
			style.Render(item.Title),
		)
		b.WriteString(line)
		b.WriteString("\n")

		// Description
		if i == m.homeState.cursor {
			b.WriteString(fmt.Sprintf("      %s", m.styles.Muted.Render(item.Description)))
			b.WriteString("\n")
		}
	}

	return m.styles.Content.Render(b.String())
}

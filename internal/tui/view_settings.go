package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/ui"
)

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.settingsState.cursor--
		if m.settingsState.cursor < 0 {
			m.settingsState.cursor = 0
		}

	case key.Matches(msg, m.keys.Down):
		m.settingsState.cursor++
		maxCursor := 0 // Only theme setting for now
		if m.settingsState.cursor > maxCursor {
			m.settingsState.cursor = maxCursor
		}

	case key.Matches(msg, m.keys.Left):
		if m.settingsState.cursor == 0 { // Theme
			m.settingsState.themeIndex--
			if m.settingsState.themeIndex < 0 {
				m.settingsState.themeIndex = len(m.settingsState.themeOptions) - 1
			}
			return m.applyTheme()
		}

	case key.Matches(msg, m.keys.Right):
		if m.settingsState.cursor == 0 { // Theme
			m.settingsState.themeIndex++
			if m.settingsState.themeIndex >= len(m.settingsState.themeOptions) {
				m.settingsState.themeIndex = 0
			}
			return m.applyTheme()
		}

	case key.Matches(msg, m.keys.Theme):
		// Cycle through themes
		m.settingsState.themeIndex++
		if m.settingsState.themeIndex >= len(m.settingsState.themeOptions) {
			m.settingsState.themeIndex = 0
		}
		return m.applyTheme()
	}

	return m, nil
}

func (m Model) applyTheme() (tea.Model, tea.Cmd) {
	themeName := m.settingsState.themeOptions[m.settingsState.themeIndex]
	ui.SetTheme(themeName)
	m.styles.RefreshStyles()
	return m, nil
}

func (m Model) viewSettings() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Settings"))
	b.WriteString("\n\n")

	// Theme setting
	cursor := "  "
	if m.settingsState.cursor == 0 {
		cursor = m.styles.Primary.Render("> ")
	}

	themeName := m.settingsState.themeOptions[m.settingsState.themeIndex]
	themeDisplay := fmt.Sprintf("< %s >", themeName)

	b.WriteString(fmt.Sprintf("%sTheme: %s\n",
		cursor,
		m.styles.Highlight.Render(themeDisplay)))

	b.WriteString("\n")

	// Current configuration display
	configBox := m.styles.Box.Render(
		fmt.Sprintf("%s\n\n%s %d\n%s %s\n%s %d\n%s %d",
			m.styles.BoxTitle.Render("Current Configuration"),
			m.styles.ResultLabel.Render("Default N:"),
			m.config.N,
			m.styles.ResultLabel.Render("Timeout:"),
			m.config.Timeout.String(),
			m.styles.ResultLabel.Render("Parallel Threshold:"),
			m.config.Threshold,
			m.styles.ResultLabel.Render("FFT Threshold:"),
			m.config.FFTThreshold,
		),
	)
	b.WriteString(configBox)
	b.WriteString("\n\n")

	b.WriteString(m.styles.Muted.Render("←/→: Change value | t: Cycle theme | Esc: Back"))

	return m.styles.Content.Render(b.String())
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.helpState.scrollOffset > 0 {
			m.helpState.scrollOffset--
		}

	case key.Matches(msg, m.keys.Down):
		m.helpState.scrollOffset++
	}

	return m, nil
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Help & Keyboard Shortcuts"))
	b.WriteString("\n\n")

	sections := []struct {
		title    string
		bindings []struct {
			keys string
			desc string
		}
	}{
		{
			title: "Navigation",
			bindings: []struct {
				keys string
				desc string
			}{
				{"↑/k, ↓/j", "Move up/down"},
				{"←/h, →/l", "Move left/right"},
				{"Tab", "Next field"},
				{"Shift+Tab", "Previous field"},
				{"Enter", "Confirm/Select"},
				{"Esc", "Back/Cancel"},
			},
		},
		{
			title: "Actions",
			bindings: []struct {
				keys string
				desc string
			}{
				{"c", "New calculation"},
				{"m", "Compare all algorithms"},
				{"t", "Change theme"},
				{"s", "Open settings"},
				{"x", "Toggle hexadecimal display"},
				{"v", "Toggle full value display"},
			},
		},
		{
			title: "General",
			bindings: []struct {
				keys string
				desc string
			}{
				{"?/F1", "Show/hide help"},
				{"q/Ctrl+C", "Quit application"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(m.styles.BoxTitle.Render(section.title))
		b.WriteString("\n")

		for _, binding := range section.bindings {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				m.styles.HelpKey.Render(fmt.Sprintf("%-15s", binding.keys)),
				m.styles.HelpDesc.Render(binding.desc)))
		}
		b.WriteString("\n")
	}

	// About section
	b.WriteString(m.styles.BoxTitle.Render("About"))
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("  Fibonacci Calculator TUI"))
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("  High-performance Fibonacci number calculation"))
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("  Supports multiple algorithms: Fast Doubling, Matrix, FFT"))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Muted.Render("Press Esc or ? to close help"))

	return m.styles.Content.Render(b.String())
}

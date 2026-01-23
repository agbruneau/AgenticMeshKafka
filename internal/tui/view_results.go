package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/orchestration"
)

func (m Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.NewCalc):
		m.currentView = ViewCalculator
		return m, nil

	case key.Matches(msg, m.keys.Compare):
		return m.startComparison()

	case key.Matches(msg, m.keys.HexToggle):
		m.resultsState.showHex = !m.resultsState.showHex
		return m, nil

	case key.Matches(msg, m.keys.SaveResult):
		return m.saveResultToFile()

	case msg.String() == "v":
		m.resultsState.showFull = !m.resultsState.showFull
		return m, nil
	}

	return m, nil
}

func (m Model) saveResultToFile() (tea.Model, tea.Cmd) {
	if m.resultsState.result == nil || m.resultsState.result.Result == nil {
		m.lastError = fmt.Errorf("no result to save")
		return m, nil
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("fibonacci_%d_%s.txt", m.resultsState.n, timestamp)

	// Get user's home directory or current directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	filepath := filepath.Join(homeDir, filename)

	// Build file content
	var content strings.Builder
	content.WriteString("Fibonacci Calculator Result\n")
	content.WriteString("===========================\n\n")
	fmt.Fprintf(&content, "Index: F(%d)\n", m.resultsState.n)
	fmt.Fprintf(&content, "Algorithm: %s\n", m.resultsState.result.Name)
	fmt.Fprintf(&content, "Duration: %s\n", m.resultsState.result.Duration)
	fmt.Fprintf(&content, "Digits: %d\n\n", len(m.resultsState.result.Result.String()))
	fmt.Fprintf(&content, "Value:\n%s\n", m.resultsState.result.Result.String())

	if m.resultsState.showHex {
		fmt.Fprintf(&content, "\nHexadecimal:\n0x%s\n", m.resultsState.result.Result.Text(16))
	}

	// Write to file (0o600 permissions for security)
	if err := os.WriteFile(filepath, []byte(content.String()), 0o600); err != nil {
		m.lastError = fmt.Errorf("failed to save: %v", err)
		return m, nil
	}

	m.lastError = nil
	// Show success message via a temporary message
	return m, func() tea.Msg {
		return ResultSavedMsg{Path: filepath}
	}
}

// ResultSavedMsg indicates a result was saved successfully.
type ResultSavedMsg struct {
	Path string
}

func (m Model) viewResults() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Calculation Complete"))
	b.WriteString("\n\n")

	if m.resultsState.result == nil {
		b.WriteString(m.styles.Error.Render("No result available"))
		return m.styles.Content.Render(b.String())
	}

	result := m.resultsState.result

	if result.Err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", result.Err)))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Muted.Render("Press 'c' for new calculation | Esc to go back"))
		return m.styles.Content.Render(b.String())
	}

	// Result summary
	summaryBox := m.styles.Box.Render(
		fmt.Sprintf("%s\n\n%s %s\n%s %s\n%s %s",
			m.styles.BoxTitle.Render("Result Summary"),
			m.styles.ResultLabel.Render("Index:"),
			m.styles.Highlight.Render(fmt.Sprintf("F(%d)", m.resultsState.n)),
			m.styles.ResultLabel.Render("Algorithm:"),
			m.styles.Info.Render(result.Name),
			m.styles.ResultLabel.Render("Duration:"),
			m.styles.Success.Render(result.Duration.String()),
		),
	)
	b.WriteString(summaryBox)
	b.WriteString("\n\n")

	// Result value
	if result.Result != nil {
		m.renderResultValue(&b, result)
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("c: New | m: Compare | x: Hex | v: Full | Ctrl+S: Save | Esc: Home"))

	return m.styles.Content.Render(b.String())
}

func (m Model) renderResultValue(b *strings.Builder, result *orchestration.CalculationResult) {
	valueStr := result.Result.String()
	if m.resultsState.showHex {
		valueStr = "0x" + result.Result.Text(16)
	}

	numDigits := len(result.Result.String())
	fmt.Fprintf(b, "%s %s digits\n\n",
		m.styles.ResultLabel.Render("Length:"),
		m.styles.Highlight.Render(fmt.Sprintf("%d", numDigits)))

	// Truncate if too long based on terminal width
	maxLen := m.getMaxValueLength()
	displayValue := valueStr
	if !m.resultsState.showFull && len(valueStr) > maxLen {
		half := maxLen / 2
		displayValue = valueStr[:half] + "..." + valueStr[len(valueStr)-half:]
	}

	b.WriteString(m.styles.ResultLabel.Render("Value:"))
	b.WriteString("\n")
	b.WriteString(m.styles.ResultValue.Render(displayValue))
	b.WriteString("\n")
}

func (m Model) getMaxValueLength() int {
	// Adapt display length based on terminal width
	if m.width > 120 {
		return 200
	}
	if m.width > 80 {
		return 100
	}
	return 60
}

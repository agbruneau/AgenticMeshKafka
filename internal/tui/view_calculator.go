package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/fibonacci"
)

const (
	fieldN = iota
	fieldAlgo
	fieldStart
)

func (m Model) updateCalculator(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.calculatorState.focusedField == fieldAlgo {
			m.calculatorState.selectedAlgo--
			if m.calculatorState.selectedAlgo < 0 {
				m.calculatorState.selectedAlgo = len(m.calculatorState.availableAlgos) - 1
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.calculatorState.focusedField == fieldAlgo {
			m.calculatorState.selectedAlgo++
			if m.calculatorState.selectedAlgo >= len(m.calculatorState.availableAlgos) {
				m.calculatorState.selectedAlgo = 0
			}
		}

	case key.Matches(msg, m.keys.Tab):
		m.calculatorState.focusedField++
		if m.calculatorState.focusedField > fieldStart {
			m.calculatorState.focusedField = fieldN
		}

	case key.Matches(msg, m.keys.ShiftTab):
		m.calculatorState.focusedField--
		if m.calculatorState.focusedField < fieldN {
			m.calculatorState.focusedField = fieldStart
		}

	case key.Matches(msg, m.keys.Enter):
		if m.calculatorState.focusedField == fieldStart {
			return m.startSingleCalculation()
		}
		// Move to next field
		m.calculatorState.focusedField++
		if m.calculatorState.focusedField > fieldStart {
			m.calculatorState.focusedField = fieldN
		}

	default:
		// Handle numeric input for N field
		if m.calculatorState.focusedField == fieldN {
			return m.handleNumericInput(msg)
		}
	}

	return m, nil
}

func (m Model) handleNumericInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	switch keyStr {
	case "backspace":
		if m.calculatorState.inputN != "" {
			m.calculatorState.inputN = m.calculatorState.inputN[:len(m.calculatorState.inputN)-1]
		}
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.calculatorState.inputN += keyStr
	}

	return m, nil
}

func (m Model) startSingleCalculation() (tea.Model, tea.Cmd) {
	// Parse N
	n, err := strconv.ParseUint(m.calculatorState.inputN, 10, 64)
	if err != nil || n == 0 {
		m.lastError = fmt.Errorf("invalid value for N: %s", m.calculatorState.inputN)
		return m, nil
	}

	// Get selected algorithm
	algoName := m.calculatorState.availableAlgos[m.calculatorState.selectedAlgo]

	// Find calculator(s)
	var calculator fibonacci.Calculator
	if algoName == "all" {
		// Start comparison instead
		m.config.N = n
		return m.startComparison()
	}

	for _, c := range m.calculators {
		if c.Name() == algoName {
			calculator = c
			break
		}
	}

	if calculator == nil {
		m.lastError = fmt.Errorf("calculator not found: %s", algoName)
		return m, nil
	}

	// Set up progress tracking
	m.progressState = ProgressState{
		n:              n,
		algorithm:      algoName,
		numCalculators: 1,
		progresses:     []float64{0},
		startTime:      time.Now().UnixNano(),
		done:           false,
		progressChan:   make(chan fibonacci.ProgressUpdate, 10),
	}

	// Switch to progress view
	m.currentView = ViewProgress
	m.lastError = nil

	// Start calculation
	opts := m.config.ToCalculationOptions()
	return m, tea.Batch(
		runCalculation(m.ctx, calculator, n, opts, m.progressState.progressChan, 0),
		listenForProgress(m.progressState.progressChan),
	)
}

func (m Model) viewCalculator() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Calculate Fibonacci Number"))
	b.WriteString("\n\n")

	// N input field
	nLabel := "Fibonacci Index (N):"
	nStyle := m.styles.Input
	if m.calculatorState.focusedField == fieldN {
		nLabel = "> " + nLabel
		nStyle = m.styles.InputFocused
	} else {
		nLabel = "  " + nLabel
	}
	b.WriteString(nLabel)
	b.WriteString("\n")
	inputDisplay := m.calculatorState.inputN
	if inputDisplay == "" {
		inputDisplay = "0"
	}
	b.WriteString(nStyle.Render(fmt.Sprintf("  %s", inputDisplay)))
	b.WriteString("\n\n")

	// Algorithm selector
	algoLabel := "Algorithm:"
	if m.calculatorState.focusedField == fieldAlgo {
		algoLabel = "> " + algoLabel
	} else {
		algoLabel = "  " + algoLabel
	}
	b.WriteString(algoLabel)
	b.WriteString("\n")

	for i, algo := range m.calculatorState.availableAlgos {
		cursor := "    "
		style := m.styles.MenuItem
		if i == m.calculatorState.selectedAlgo {
			if m.calculatorState.focusedField == fieldAlgo {
				cursor = m.styles.Primary.Render("  > ")
				style = m.styles.MenuItemActive
			} else {
				cursor = "  * "
				style = m.styles.Highlight
			}
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(algo)))
	}
	b.WriteString("\n")

	// Start button
	startLabel := "[Start Calculation]"
	startStyle := m.styles.Button
	if m.calculatorState.focusedField == fieldStart {
		startStyle = m.styles.ButtonFocused
		startLabel = "> " + startLabel
	} else {
		startLabel = "  " + startLabel
	}
	b.WriteString(startStyle.Render(startLabel))
	b.WriteString("\n\n")

	// Help text
	b.WriteString(m.styles.Muted.Render("Tab: Next field | Enter: Confirm | Esc: Back"))

	return m.styles.Content.Render(b.String())
}

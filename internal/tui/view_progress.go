package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/cli"
)

func (m Model) updateProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only Escape is allowed during progress (to cancel)
	if key.Matches(msg, m.keys.Escape) {
		m.cancel()
		m.ctx, m.cancel = context.WithCancel(context.Background())
		m.currentView = ViewCalculator
		m.progressState.done = true
	}
	return m, nil
}

func (m Model) viewProgress() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Calculating..."))
	b.WriteString("\n\n")

	// Show what's being calculated
	b.WriteString(fmt.Sprintf("Computing F(%s)\n",
		m.styles.Highlight.Render(fmt.Sprintf("%d", m.progressState.n))))
	b.WriteString(fmt.Sprintf("Algorithm: %s\n\n",
		m.styles.Info.Render(m.progressState.algorithm)))

	// Progress bar
	if len(m.progressState.progresses) > 0 {
		progress := m.progressState.progresses[0]
		barWidth := 40

		// Calculate ETA
		elapsed := time.Since(time.Unix(0, m.progressState.startTime))
		var eta time.Duration
		if progress > 0.01 && elapsed > 100*time.Millisecond {
			remaining := 1.0 - progress
			etaSeconds := (elapsed.Seconds() / progress) * remaining
			eta = time.Duration(etaSeconds * float64(time.Second))
		}

		// Render progress bar
		bar := renderProgressBar(progress, barWidth, m.styles)
		b.WriteString(bar)
		b.WriteString("\n")

		// Progress percentage and ETA
		b.WriteString(fmt.Sprintf("%s%.1f%%",
			m.styles.Highlight.Render(""),
			progress*100))

		if eta > 0 {
			b.WriteString(fmt.Sprintf("  ETA: %s",
				m.styles.Muted.Render(cli.FormatETA(eta))))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("Press Esc to cancel"))

	return m.styles.Content.Render(b.String())
}

func renderProgressBar(progress float64, width int, styles Styles) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", empty)

	return fmt.Sprintf("[%s%s]",
		styles.ProgressFilled.Render(filledStr),
		styles.ProgressEmpty.Render(emptyStr))
}

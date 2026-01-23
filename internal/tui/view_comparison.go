package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
)

func (m Model) startComparison() (tea.Model, tea.Cmd) {
	// Set up progress tracking for all calculators
	numCalcs := len(m.calculators)
	m.comparisonState = ComparisonState{
		n:            m.config.N,
		progresses:   make([]float64, numCalcs),
		progressChan: make(chan fibonacci.ProgressUpdate, numCalcs*10),
		inProgress:   true,
	}

	m.currentView = ViewProgress
	m.progressState = ProgressState{
		n:              m.config.N,
		algorithm:      "all",
		numCalculators: numCalcs,
		progresses:     make([]float64, numCalcs),
		startTime:      time.Now().UnixNano(),
		progressChan:   m.comparisonState.progressChan,
	}

	m.lastError = nil

	// Start comparison
	return m, tea.Batch(
		runComparison(m.ctx, m.calculators, m.config, m.comparisonState.progressChan),
		listenForProgress(m.comparisonState.progressChan),
	)
}

func (m Model) updateComparison(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.comparisonState.cursor > 0 {
			m.comparisonState.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.comparisonState.cursor < len(m.comparisonState.results)-1 {
			m.comparisonState.cursor++
		}

	case key.Matches(msg, m.keys.NewCalc):
		m.currentView = ViewCalculator
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		// Show details for selected result
		m.comparisonState.showDetails = !m.comparisonState.showDetails
		return m, nil
	}

	return m, nil
}

// comparisonResultRow holds formatted data for a single comparison result.
type comparisonResultRow struct {
	name     string
	duration string
	status   string
	isError  bool
}

func (m Model) viewComparison() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Algorithm Comparison"))
	b.WriteString("\n\n")

	if m.comparisonState.inProgress {
		m.renderComparisonProgress(&b)
		return m.styles.Content.Render(b.String())
	}

	if len(m.comparisonState.results) == 0 {
		b.WriteString(m.styles.Error.Render("No results available"))
		return m.styles.Content.Render(b.String())
	}

	sortedIndices := m.getSortedResultIndices()
	rows := m.buildResultRows(sortedIndices)

	m.renderComparisonHeader(&b, len(rows))
	m.renderComparisonTable(&b, rows)
	m.renderConsistencyCheck(&b)
	m.renderComparisonDetails(&b, sortedIndices)

	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("Enter: Toggle details | c: New calculation | Esc: Home"))

	return m.styles.Content.Render(b.String())
}

func (m Model) renderComparisonProgress(b *strings.Builder) {
	b.WriteString(m.styles.Info.Render("Running comparison..."))
	b.WriteString("\n\n")

	for i, calc := range m.calculators {
		progress := float64(0)
		if i < len(m.comparisonState.progresses) {
			progress = m.comparisonState.progresses[i]
		}
		bar := renderProgressBar(progress, 30, m.styles)
		fmt.Fprintf(b, "%s: %s %.1f%%\n",
			m.styles.Highlight.Render(fmt.Sprintf("%-12s", calc.Name())),
			bar,
			progress*100)
	}
}

func (m Model) getSortedResultIndices() []int {
	indices := make([]int, len(m.comparisonState.results))
	for i := range indices {
		indices[i] = i
	}

	results := m.comparisonState.results
	sort.Slice(indices, func(i, j int) bool {
		return compareResults(results[indices[i]], results[indices[j]])
	})

	return indices
}

func compareResults(a, b orchestration.CalculationResult) bool {
	if a.Err != nil && b.Err == nil {
		return false
	}
	if a.Err == nil && b.Err != nil {
		return true
	}
	return a.Duration < b.Duration
}

func (m Model) buildResultRows(sortedIndices []int) []comparisonResultRow {
	rows := make([]comparisonResultRow, len(sortedIndices))
	for i, idx := range sortedIndices {
		r := m.comparisonState.results[idx]
		status := m.styles.Success.Render("OK")
		if r.Err != nil {
			status = m.styles.Error.Render("ERR")
		}
		rows[i] = comparisonResultRow{
			name:     r.Name,
			duration: r.Duration.String(),
			status:   status,
			isError:  r.Err != nil,
		}
	}
	return rows
}

func (m Model) renderComparisonHeader(b *strings.Builder, count int) {
	fmt.Fprintf(b, "Calculated F(%s) with %d algorithms\n\n",
		m.styles.Highlight.Render(fmt.Sprintf("%d", m.comparisonState.n)),
		count)

	b.WriteString(m.styles.TableHeader.Render(fmt.Sprintf("  %-3s %-15s %-15s %s", "#", "Algorithm", "Duration", "Status")))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
}

func (m Model) renderComparisonTable(b *strings.Builder, rows []comparisonResultRow) {
	for i, r := range rows {
		cursor := "  "
		rowStyle := m.styles.TableRow
		if i == m.comparisonState.cursor {
			cursor = m.styles.Primary.Render("> ")
			rowStyle = m.styles.MenuItemActive
		}

		rank := m.formatRank(i, r.isError)
		line := fmt.Sprintf("%s%-3s %-15s %-15s %s",
			cursor, rank, rowStyle.Render(r.name), r.duration, r.status)
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m Model) formatRank(index int, isError bool) string {
	if index == 0 && !isError {
		return m.styles.Success.Render("1")
	}
	return fmt.Sprintf("%d", index+1)
}

func (m Model) renderComparisonDetails(b *strings.Builder, sortedIndices []int) {
	if !m.comparisonState.showDetails || m.comparisonState.cursor >= len(m.comparisonState.results) {
		return
	}

	b.WriteString("\n")
	b.WriteString(m.styles.BoxTitle.Render("Details"))
	b.WriteString("\n")

	r := m.comparisonState.results[sortedIndices[m.comparisonState.cursor]]
	if r.Err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", r.Err)))
	} else if r.Result != nil {
		digits := len(r.Result.String())
		fmt.Fprintf(b, "Result length: %d digits\n", digits)
	}
}

// checkResultsConsistency verifies that all successful calculations return the same result.
func (m Model) checkResultsConsistency() (consistent bool, message string) {
	var firstResult *orchestration.CalculationResult
	for i := range m.comparisonState.results {
		r := &m.comparisonState.results[i]
		if r.Err == nil && r.Result != nil {
			if firstResult == nil {
				firstResult = r
			} else if r.Result.Cmp(firstResult.Result) != 0 {
				return false, fmt.Sprintf("Inconsistency: %s != %s", r.Name, firstResult.Name)
			}
		}
	}
	return true, ""
}

func (m Model) renderConsistencyCheck(b *strings.Builder) {
	consistent, msg := m.checkResultsConsistency()
	b.WriteString("\n")
	if consistent {
		b.WriteString(m.styles.Success.Render("All results are consistent"))
	} else {
		b.WriteString(m.styles.Error.Render("WARNING: " + msg))
	}
	b.WriteString("\n")
}

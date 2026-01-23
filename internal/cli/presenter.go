package cli

import (
	"fmt"
	"io"
	"sync"
	"text/tabwriter"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/ui"
)

// CLIProgressReporter implements orchestration.ProgressReporter for CLI output.
// It wraps the DisplayProgress function to provide a spinner and progress bar
// display during calculations.
type CLIProgressReporter struct{}

// Verify that CLIProgressReporter implements orchestration.ProgressReporter.
var _ orchestration.ProgressReporter = CLIProgressReporter{}

// DisplayProgress displays a spinner and progress bar for ongoing calculations.
func (CLIProgressReporter) DisplayProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, out io.Writer) {
	DisplayProgress(wg, progressChan, numCalculators, out)
}

// CLIResultPresenter implements orchestration.ResultPresenter for CLI output.
// It provides formatted, colorized output for calculation results in the
// command-line interface.
type CLIResultPresenter struct{}

// Verify that CLIResultPresenter implements orchestration.ResultPresenter.
var _ orchestration.ResultPresenter = CLIResultPresenter{}

// PresentComparisonTable displays the comparison summary table with
// algorithm names, durations, and status in a formatted tabular layout.
func (CLIResultPresenter) PresentComparisonTable(results []orchestration.CalculationResult, out io.Writer) {
	fmt.Fprintf(out, "\n--- Comparison Summary ---\n")
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
		ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset())

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", ui.ColorRed(), res.Err, ui.ColorReset())
		} else {
			status = fmt.Sprintf("%s✅ Success%s", ui.ColorGreen(), ui.ColorReset())
		}
		duration := FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			duration = "< 1µs"
		}
		fmt.Fprintf(tw, "%s%s%s\t%s%s%s\t%s\n",
			ui.ColorBlue(), res.Name, ui.ColorReset(),
			ui.ColorYellow(), duration, ui.ColorReset(),
			status)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(out, "Warning: failed to flush tabwriter: %v\n", err)
	}
}

// PresentResult displays the final calculation result using the CLI's
// DisplayResult function.
func (CLIResultPresenter) PresentResult(result orchestration.CalculationResult, n uint64, verbose, details, concise bool, out io.Writer) {
	DisplayResult(result.Result, n, result.Duration, verbose, details, concise, out)
}

// FormatDuration formats a duration for display using the CLI's standard
// duration formatting.
func (CLIResultPresenter) FormatDuration(d time.Duration) string {
	return FormatExecutionDuration(d)
}

// HandleError handles calculation errors and returns an appropriate exit code.
func (CLIResultPresenter) HandleError(err error, duration time.Duration, out io.Writer) int {
	return apperrors.HandleCalculationError(err, duration, out, CLIColorProvider{})
}

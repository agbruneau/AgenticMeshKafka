package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/ui"
)

// Model is the root model for the TUI application.
type Model struct {
	// Configuration
	config      config.AppConfig
	calculators []fibonacci.Calculator
	ctx         context.Context
	cancel      context.CancelFunc

	// UI state
	currentView View
	prevView    View
	keys        KeyMap
	styles      Styles
	help        help.Model
	width       int
	height      int
	ready       bool

	// View-specific state
	homeState       HomeState
	calculatorState CalculatorState
	progressState   ProgressState
	resultsState    ResultsState
	comparisonState ComparisonState
	settingsState   SettingsState
	helpState       HelpState

	// Error handling
	lastError error
}

// HomeState holds state for the home view.
type HomeState struct {
	cursor int
}

// CalculatorState holds state for the calculator view.
type CalculatorState struct {
	inputN         string
	selectedAlgo   int
	focusedField   int
	availableAlgos []string
}

// ProgressState holds state for the progress view.
type ProgressState struct {
	n              uint64
	algorithm      string
	numCalculators int
	progresses     []float64
	startTime      int64
	done           bool
	progressChan   chan fibonacci.ProgressUpdate
}

// ResultsState holds state for the results view.
type ResultsState struct {
	result   *orchestration.CalculationResult
	n        uint64
	showHex  bool
	showFull bool
}

// ComparisonState holds state for the comparison view.
type ComparisonState struct {
	results      []orchestration.CalculationResult
	n            uint64
	cursor       int
	showDetails  bool
	progressChan chan fibonacci.ProgressUpdate
	progresses   []float64
	inProgress   bool
}

// SettingsState holds state for the settings view.
type SettingsState struct {
	cursor       int
	themeIndex   int
	themeOptions []string
}

// HelpState holds state for the help view.
type HelpState struct {
	scrollOffset int
}

// NewModel creates a new TUI model with the given configuration.
func NewModel(cfg config.AppConfig, calculators []fibonacci.Calculator) Model {
	ctx, cancel := context.WithCancel(context.Background())

	// Get algorithm names
	algoNames := make([]string, len(calculators))
	for i, c := range calculators {
		algoNames[i] = c.Name()
	}

	return Model{
		config:      cfg,
		calculators: calculators,
		ctx:         ctx,
		cancel:      cancel,
		currentView: ViewHome,
		keys:        DefaultKeyMap(),
		styles:      DefaultStyles(),
		help:        help.New(),
		homeState: HomeState{
			cursor: 0,
		},
		calculatorState: CalculatorState{
			inputN:         fmt.Sprintf("%d", cfg.N),
			selectedAlgo:   0,
			availableAlgos: append([]string{"all"}, algoNames...),
		},
		settingsState: SettingsState{
			themeOptions: []string{"dark", "light", "none"},
			themeIndex:   0,
		},
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(100),
	)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case NavigateMsg:
		return m.handleNavigateMsg(msg)
	case ProgressMsg:
		return m.handleProgressUpdate(msg)
	case ProgressDoneMsg:
		return m, nil
	case CalculationResultMsg:
		return m.handleCalculationResult(msg)
	case ComparisonResultsMsg:
		return m.handleComparisonResults(msg)
	case ErrorMsg:
		m.lastError = msg.Err
		return m, nil
	case ThemeChangedMsg:
		return m.handleThemeChanged(msg)
	case ResultSavedMsg:
		// Clear any error and show success (error field used for messages)
		m.lastError = nil
		return m, nil
	case TickMsg:
		return m, tickCmd(100)
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.cancel()
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		return m.toggleHelp()
	case key.Matches(msg, m.keys.Escape):
		return m.handleEscape()
	}
	return m.updateCurrentView(msg)
}

func (m Model) toggleHelp() (tea.Model, tea.Cmd) {
	if m.currentView != ViewHelp {
		m.prevView = m.currentView
		m.currentView = ViewHelp
	} else {
		m.currentView = m.prevView
	}
	return m, nil
}

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	m.help.Width = msg.Width
	return m, nil
}

func (m Model) handleNavigateMsg(msg NavigateMsg) (tea.Model, tea.Cmd) {
	m.prevView = m.currentView
	m.currentView = msg.To
	return m, nil
}

func (m Model) handleCalculationResult(msg CalculationResultMsg) (tea.Model, tea.Cmd) {
	m.resultsState.result = &msg.Result
	m.resultsState.n = msg.N
	m.progressState.done = true
	m.currentView = ViewResults
	return m, nil
}

func (m Model) handleComparisonResults(msg ComparisonResultsMsg) (tea.Model, tea.Cmd) {
	m.comparisonState.results = msg.Results
	m.comparisonState.n = msg.N
	m.comparisonState.inProgress = false
	m.currentView = ViewComparison
	return m, nil
}

func (m Model) handleThemeChanged(msg ThemeChangedMsg) (tea.Model, tea.Cmd) {
	ui.SetTheme(msg.ThemeName)
	m.styles.RefreshStyles()
	return m, nil
}

func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewHome:
		m.cancel()
		return m, tea.Quit
	case ViewHelp:
		m.currentView = m.prevView
	case ViewProgress:
		// Cancel current calculation
		m.cancel()
		m.ctx, m.cancel = context.WithCancel(context.Background())
		m.currentView = ViewCalculator
	default:
		m.currentView = ViewHome
	}
	return m, nil
}

func (m Model) updateCurrentView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewHome:
		return m.updateHome(msg)
	case ViewCalculator:
		return m.updateCalculator(msg)
	case ViewProgress:
		return m.updateProgress(msg)
	case ViewResults:
		return m.updateResults(msg)
	case ViewComparison:
		return m.updateComparison(msg)
	case ViewSettings:
		return m.updateSettings(msg)
	case ViewHelp:
		return m.updateHelp(msg)
	}
	return m, nil
}

func (m Model) handleProgressUpdate(msg ProgressMsg) (tea.Model, tea.Cmd) {
	idx := msg.Update.CalculatorIndex
	if idx >= 0 && idx < len(m.progressState.progresses) {
		m.progressState.progresses[idx] = msg.Update.Value
	}
	if idx >= 0 && idx < len(m.comparisonState.progresses) {
		m.comparisonState.progresses[idx] = msg.Update.Value
	}
	// Continue listening for progress
	if m.progressState.progressChan != nil {
		return m, listenForProgress(m.progressState.progressChan)
	}
	if m.comparisonState.progressChan != nil && m.comparisonState.inProgress {
		return m, listenForProgress(m.comparisonState.progressChan)
	}
	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content string
	switch m.currentView {
	case ViewHome:
		content = m.viewHome()
	case ViewCalculator:
		content = m.viewCalculator()
	case ViewProgress:
		content = m.viewProgress()
	case ViewResults:
		content = m.viewResults()
	case ViewComparison:
		content = m.viewComparison()
	case ViewSettings:
		content = m.viewSettings()
	case ViewHelp:
		content = m.viewHelp()
	}

	// Build the full view with header and footer
	return m.buildFrame(content)
}

func (m Model) buildFrame(content string) string {
	// Header
	header := m.styles.Header.Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			m.styles.Title.Render("Fibonacci Calculator"),
			strings.Repeat(" ", maxInt(0, m.width-40)),
			m.styles.Muted.Render(fmt.Sprintf("Theme: %s", ui.GetCurrentTheme().Name)),
		),
	)

	// Footer with help
	footer := m.styles.Footer.Render(
		m.help.ShortHelpView(m.keys.ShortHelp()),
	)

	// Error display
	var errorBar string
	if m.lastError != nil {
		errorBar = m.styles.Error.Render(fmt.Sprintf("Error: %v", m.lastError))
	}

	// Combine all parts
	parts := []string{header}
	if errorBar != "" {
		parts = append(parts, errorBar)
	}
	parts = append(parts, content, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

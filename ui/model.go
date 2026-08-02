package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/filesystem"
)

// Pane represents a distinct section of the UI.
type Pane int

const (
	// PaneExplorer is the file explorer pane.
	PaneExplorer Pane = iota
	// PaneOutput is the test output pane.
	PaneOutput
)

// LeftTab represents the active tab in the left pane.
type LeftTab int

const (
	// TabExplorer is the file explorer tab.
	TabExplorer LeftTab = iota
	// TabWatched is the watched files tab.
	TabWatched
)

// DisplayNode represents a node in the explorer list, potentially compacted.
type DisplayNode struct {
	*filesystem.Node
	DisplayName string
	Depth       int
}

// Model represents the application state for the Bubbletea program.
type Model struct {
	// UI State
	activePane Pane
	width      int
	height     int
	ready      bool
	showHelp   bool
	cursor     int
	viewport   viewport.Model

	// Tab State
	activeTab     LeftTab
	watchedCursor int

	// Search State
	searchMode        bool
	searchFocus       bool
	searchInput       textinput.Model
	searchMatches     []int
	currentMatchIndex int

	// Components
	keys KeyMap
	help help.Model

	// Mouse State
	lastClickTime time.Time
	lastClickX    int
	lastClickY    int

	// Notification State
	activeNotification  string
	isNotificationError bool
	notificationID      int

	// Data / Dependencies
	engine    *engine.Engine
	flatNodes []DisplayNode
}

// NewModel creates and initializes a new Model.
func NewModel(eng *engine.Engine) Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#A0A0A0"})
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#808080"})
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#606060"})
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#A0A0A0"})
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#808080"})
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#606060"})
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "/"
	ti.CharLimit = 156
	ti.Width = 20

	return Model{
		activePane:  PaneExplorer,
		engine:      eng,
		keys:        NewKeyMap(),
		help:        h,
		searchInput: ti,
	}
}

// Init initializes the Bubbletea program.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.engine.Init(),
		tea.EnableMouseCellMotion,
	)
}

// View renders the UI based on the current state.
func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	if m.width == 0 {
		return "Loading..."
	}

	paneWidth := (m.width / 2) - 2
	paneHeight := m.height - 4

	// Explorer View
	explorerRender := m.renderExplorer(paneWidth, paneHeight)

	// Output View
	var outputView strings.Builder
	if m.engine.IsSmartMode() {
		passed, failed, running := m.engine.GetSuiteStats()
		badge := m.renderSuiteBadge(passed, failed, running)
		outputView.WriteString(badge + "\n")
	} else {
		outputView.WriteString(titleStyle.Render("OUTPUT") + "\n\n")
	}

	if !m.ready {
		outputView.WriteString("Initializing...")
	} else {
		outputView.WriteString(m.viewport.View())
	}

	outputStyle := paneStyle
	if m.activePane == PaneOutput {
		outputStyle = activePaneStyle
	}
	outputRender := outputStyle.
		Width(paneWidth).
		Height(paneHeight).
		Render(outputView.String())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, explorerRender, outputRender)
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, panes, footer)
}

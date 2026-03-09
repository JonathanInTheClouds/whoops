package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/JonathanInTheClouds/whoops/internal/git"
)

type mode int

const (
	modeNormal mode = iota
	modeConfirm
	modeDone
	modeError
)

// Model is the Bubble Tea model for the --history TUI.
type Model struct {
	actions  []git.Action
	cursor   int
	mode     mode
	dryRun   bool
	result   *git.UndoResult
	errMsg   string
	viewport viewport.Model
	width    int
	height   int
}

// Messages
type actionsLoadedMsg struct{ actions []git.Action }
type undoneMsg struct{ result *git.UndoResult }
type errMsg struct{ err error }

func NewModel(dryRun bool) Model {
	return Model{
		dryRun: dryRun,
		mode:   modeNormal,
	}
}

func (m Model) Init() tea.Cmd {
	return loadActions
}

func loadActions() tea.Msg {
	actions, err := git.ReadReflog(20)
	if err != nil {
		return errMsg{err}
	}
	return actionsLoadedMsg{actions}
}

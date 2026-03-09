package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/JonathanInTheClouds/whoops/internal/git"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(m.width, m.height-6)
		return m, nil

	case actionsLoadedMsg:
		m.actions = msg.actions
		return m, nil

	case undoneMsg:
		m.result = msg.result
		m.mode = modeDone
		return m, nil

	case errMsg:
		m.errMsg = msg.err.Error()
		m.mode = modeError
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			return m.handleNormalKeys(msg)
		case modeConfirm:
			return m.handleConfirmKeys(msg)
		case modeDone, modeError:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.actions)-1 {
			m.cursor++
		}

	case "enter":
		if len(m.actions) > 0 {
			m.mode = modeConfirm
		}
	}

	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.mode = modeNormal
		action := m.actions[m.cursor]
		return m, doUndo(action, m.dryRun)

	case "n", "N", "esc":
		m.mode = modeNormal
	}

	return m, nil
}

func doUndo(action git.Action, dryRun bool) tea.Cmd {
	return func() tea.Msg {
		result, err := git.Undo(action, dryRun)
		if err != nil {
			return errMsg{err}
		}
		return undoneMsg{result}
	}
}

var _ = viewport.New

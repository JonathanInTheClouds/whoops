package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/JonathanInTheClouds/whoops/internal/git"
)

var (
	accent  = lipgloss.Color("#A78BFA")
	green   = lipgloss.Color("#34D399")
	red     = lipgloss.Color("#F87171")
	amber   = lipgloss.Color("#F59E0B")
	dimmed  = lipgloss.Color("#9CA3AF")
	subtle  = lipgloss.Color("#6B7280")
	white   = lipgloss.Color("#F9FAFB")
	hilite  = lipgloss.Color("#7C3AED")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(hilite).
			Foreground(white).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(white).
			Padding(0, 1)

	actionTypeStyle = func(t git.ActionType) lipgloss.Style {
		color := accent
		switch t {
		case git.ActionCommit:
			color = green
		case git.ActionMerge, git.ActionRebase:
			color = amber
		case git.ActionStash, git.ActionStashPop:
			color = lipgloss.Color("#60A5FA")
		case git.ActionCheckout:
			color = lipgloss.Color("#F472B6")
		case git.ActionUnknown:
			color = dimmed
		}
		return lipgloss.NewStyle().Foreground(color).Bold(true).Width(12)
	}

	dateStyle  = lipgloss.NewStyle().Foreground(dimmed)
	helpStyle  = lipgloss.NewStyle().Foreground(subtle).Padding(0, 1)
	errorStyle = lipgloss.NewStyle().Foreground(red).Bold(true).Padding(0, 1)
	doneStyle  = lipgloss.NewStyle().Foreground(green).Bold(true).Padding(0, 1)
	dimStyle   = lipgloss.NewStyle().Foreground(dimmed).Italic(true).Padding(0, 1)
	warnStyle  = lipgloss.NewStyle().Foreground(amber).Bold(true).Padding(0, 1)
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.mode {
	case modeDone:
		return m.renderDone()
	case modeError:
		return m.renderError()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.renderList(),
		m.renderStatusBar(),
	)
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("⎌  whoops — git history")
	count := dimStyle.Render(fmt.Sprintf("%d actions", len(m.actions)))
	pad := m.width - lipgloss.Width(title) - lipgloss.Width(count)
	if pad < 0 {
		pad = 0
	}
	return title + strings.Repeat(" ", pad) + count
}

func (m Model) renderList() string {
	if len(m.actions) == 0 {
		return dimStyle.Render("No git actions found in reflog.")
	}

	var rows []string
	for i, a := range m.actions {
		typeLabel := actionTypeStyle(a.Type).Render(string(a.Type))
		desc := truncate(a.Description, m.width-30)
		date := dateStyle.Render(git.RelativeTime(a.Date))

		line := fmt.Sprintf("%s  %-*s  %s", typeLabel, m.width-30, desc, date)

		if i == m.cursor {
			rows = append(rows, selectedStyle.Render(line))
		} else {
			rows = append(rows, normalStyle.Render(line))
		}
	}

	return strings.Join(rows, "\n")
}

func (m Model) renderStatusBar() string {
	if m.mode == modeConfirm {
		action := m.actions[m.cursor]
		prefix := "Undo"
		if m.dryRun {
			prefix = "Dry run undo"
		}
		return warnStyle.Render(fmt.Sprintf("⚠ %s: %s %q? (y/n)", prefix, action.Type, truncate(action.Description, 40)))
	}
	return helpStyle.Render("↑↓ navigate   enter select   q quit")
}

func (m Model) renderDone() string {
	if m.result == nil {
		return ""
	}

	var b strings.Builder

	if m.result.DryRun {
		b.WriteString(warnStyle.Render("Dry run — nothing was changed.") + "\n\n")
		b.WriteString(dimStyle.Render("Would have run:") + "\n")
		b.WriteString(lipgloss.NewStyle().Foreground(accent).Padding(0, 2).Render(m.result.Command) + "\n\n")
		b.WriteString(dimStyle.Render("Action: ") + normalStyle.Render(string(m.result.Action.Type)))
		b.WriteString("\n" + dimStyle.Render("Description: ") + normalStyle.Render(m.result.Action.Description))
	} else {
		b.WriteString(doneStyle.Render("✓ Undone: "+string(m.result.Action.Type)) + "\n\n")
		b.WriteString(dimStyle.Render("Ran: ") + lipgloss.NewStyle().Foreground(accent).Render(m.result.Command) + "\n")
		b.WriteString(dimStyle.Render("Description: ") + normalStyle.Render(m.result.Action.Description))
	}

	b.WriteString("\n\n" + helpStyle.Render("Press any key to exit."))
	return b.String()
}

func (m Model) renderError() string {
	return errorStyle.Render("✗ "+m.errMsg) +
		"\n\n" + helpStyle.Render("Press any key to exit.")
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

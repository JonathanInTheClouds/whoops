package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ActionType represents the kind of git action that was performed.
type ActionType string

const (
	ActionCommit   ActionType = "commit"
	ActionAdd      ActionType = "add"
	ActionStash    ActionType = "stash"
	ActionStashPop ActionType = "stash pop"
	ActionMerge    ActionType = "merge"
	ActionRebase   ActionType = "rebase"
	ActionCheckout ActionType = "checkout"
	ActionUnknown  ActionType = "unknown"
)

// Action represents a single entry from the git reflog.
type Action struct {
	Type        ActionType
	Description string
	FromHash    string
	ToHash      string
	Date        time.Time
	Raw         string
	Position    int // lower = more recent, used as tiebreaker
}

// UndoResult describes what whoops did.
type UndoResult struct {
	Action  Action
	Command string
	DryRun  bool
}

// internalEntry returns true for HEAD reflog entries that are git housekeeping
// rather than user-initiated actions.
func internalEntry(subject string) bool {
	lower := strings.ToLower(strings.TrimSpace(subject))
	internals := []string{
		"reset:",        // all reset entries are internal housekeeping
		"stash: index", // internal stash index commit
	}
	for _, prefix := range internals {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// readHeadReflog reads meaningful entries from the HEAD reflog.
func readHeadReflog(limit int) ([]Action, error) {
	out, err := run("git", "reflog", "show", "--format=%H|%ct|%gs", fmt.Sprintf("-n%d", limit*4))
	if err != nil {
		return nil, err
	}

	var actions []Action
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		a, err := parseReflogLine(line)
		if err != nil {
			continue
		}
		if internalEntry(a.Raw) {
			continue
		}
		actions = append(actions, a)
		if len(actions) >= limit {
			break
		}
	}
	return actions, nil
}

// readStashReflog reads entries from the stash reflog (refs/stash).
func readStashReflog(limit int) ([]Action, error) {
	out, err := run("git", "reflog", "show", "--format=%H|%ct|%gs", fmt.Sprintf("-n%d", limit), "refs/stash")
	if err != nil {
		return nil, nil
	}

	var actions []Action
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		a, err := parseReflogLine(line)
		if err != nil {
			continue
		}
		a.Type = ActionStash
		a.Description = cleanStashDescription(a.Raw)
		actions = append(actions, a)
	}
	return actions, nil
}

// ReadReflog returns the N most recent meaningful actions, merging HEAD and
// stash reflogs sorted by date so stash operations appear in the right order.
func ReadReflog(limit int) ([]Action, error) {
	if !IsGitRepo() {
		return nil, fmt.Errorf("not inside a git repository")
	}

	headActions, err := readHeadReflog(limit)
	if err != nil {
		return nil, fmt.Errorf("could not read reflog: %w", err)
	}

	stashActions, _ := readStashReflog(limit) // ignore error — no stash is fine

	// Assign positions within each list (0 = most recent)
	for i := range headActions {
		headActions[i].Position = i
	}
	for i := range stashActions {
		stashActions[i].Position = i
	}

	// Merge and sort by date descending.
	// Tiebreak by position (lower = more recent within its own reflog).
	// Stash entries win ties over HEAD entries since a stash push always
	// comes after the reset that git internally runs on HEAD.
	all := append(headActions, stashActions...)
	sort.SliceStable(all, func(i, j int) bool {
		ti, tj := all[i].Date, all[j].Date
		if ti.Equal(tj) {
			// Stash entries win on a tie
			iIsStash := all[i].Type == ActionStash
			jIsStash := all[j].Type == ActionStash
			if iIsStash != jIsStash {
				return iIsStash
			}
			// Both same type — use position as tiebreaker
			return all[i].Position < all[j].Position
		}
		return ti.After(tj)
	})

	if len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// Undo performs the inverse of the given action.
func Undo(a Action, dryRun bool) (*UndoResult, error) {
	result := &UndoResult{Action: a, DryRun: dryRun}

	var cmd []string

	switch a.Type {
	case ActionCommit:
		// Verify HEAD~1 exists before trying
		if _, err := run("git", "rev-parse", "--verify", "HEAD~1"); err != nil {
			return nil, fmt.Errorf("no previous commit to reset to — this is the first commit")
		}
		cmd = []string{"git", "reset", "--soft", "HEAD~1"}

	case ActionStash:
		cmd = []string{"git", "stash", "pop"}

	case ActionStashPop:
		cmd = []string{"git", "stash"}

	case ActionMerge, ActionRebase:
		orig, origErr := run("git", "rev-parse", "ORIG_HEAD")
		if origErr != nil {
			return nil, fmt.Errorf("ORIG_HEAD not found — cannot undo %s", a.Type)
		}
		cmd = []string{"git", "reset", "--hard", strings.TrimSpace(orig)}

	case ActionCheckout:
		if a.FromHash != "" && a.FromHash != a.ToHash {
			cmd = []string{"git", "checkout", a.FromHash, "--"}
		} else {
			return nil, fmt.Errorf("cannot determine pre-checkout state")
		}

	case ActionAdd:
		return nil, fmt.Errorf("cannot undo git add from reflog — use `git restore --staged <file>`")

	default:
		return nil, fmt.Errorf("don't know how to undo: %s", a.Description)
	}

	result.Command = strings.Join(cmd, " ")

	if dryRun {
		return result, nil
	}

	if _, err := run(cmd[0], cmd[1:]...); err != nil {
		return nil, fmt.Errorf("undo failed: %w", err)
	}

	return result, nil
}

// IsGitRepo checks whether the current directory is inside a git repo.
func IsGitRepo() bool {
	_, err := run("git", "rev-parse", "--git-dir")
	return err == nil
}

// --- helpers ---

func parseReflogLine(line string) (Action, error) {
	// Format: <hash>|<timestamp>|<subject>
	parts := strings.SplitN(line, "|", 3)
	if len(parts) < 3 {
		return Action{}, fmt.Errorf("unexpected reflog format: %s", line)
	}

	toHash := strings.TrimSpace(parts[0])
	ts, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	subject := strings.TrimSpace(parts[2])

	date := time.Unix(ts, 0)
	actionType, description := classifySubject(subject)

	return Action{
		Type:        actionType,
		Description: description,
		FromHash:    "",
		ToHash:      toHash,
		Date:        date,
		Raw:         subject,
	}, nil
}

// classifySubject maps a reflog subject string to an ActionType.
func classifySubject(subject string) (ActionType, string) {
	lower := strings.ToLower(subject)

	switch {
	case strings.HasPrefix(lower, "commit"):
		msg := subject
		for _, prefix := range []string{"commit: ", "commit (initial): ", "commit (merge): "} {
			msg = strings.TrimPrefix(msg, prefix)
		}
		return ActionCommit, msg

	case strings.HasPrefix(lower, "stash pop") || strings.HasPrefix(lower, "stash: popping"):
		return ActionStashPop, subject

	// Note: "stash: wip on" and "stash: on" come from refs/stash reflog
	// and are handled in readStashReflog directly. Here we just catch any
	// remaining stash entries in HEAD reflog.
	case lower == "stash":
		return ActionStash, subject

	case strings.HasPrefix(lower, "merge"):
		return ActionMerge, subject

	case strings.HasPrefix(lower, "rebase"):
		return ActionRebase, subject

	case strings.HasPrefix(lower, "checkout"):
		return ActionCheckout, subject

	default:
		return ActionUnknown, subject
	}
}

// cleanStashDescription strips the commit hash from stash descriptions.
// "WIP on master: abc1234 my message" -> "WIP on master: my message"
func cleanStashDescription(raw string) string {
	// Format is typically "WIP on <branch>: <hash> <message>"
	parts := strings.SplitN(raw, ": ", 2)
	if len(parts) != 2 {
		return raw
	}
	rest := parts[1]
	// Strip leading hash if present (7+ hex chars followed by space)
	words := strings.SplitN(rest, " ", 2)
	if len(words) == 2 && len(words[0]) >= 7 && isHex(words[0]) {
		return parts[0] + ": " + words[1]
	}
	return raw
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// RelativeTime returns a human-friendly relative time string.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return strconv.Itoa(m) + " min" + plural(m) + " ago"
	case d < 24*time.Hour:
		h := int(d.Hours())
		return strconv.Itoa(h) + " hour" + plural(h) + " ago"
	case d < 7*24*time.Hour:
		day := int(d.Hours() / 24)
		return strconv.Itoa(day) + " day" + plural(day) + " ago"
	default:
		return t.Format("Jan 2, 2006")
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ZeroTime returns a zero time for use in tests.
func ZeroTime() time.Time {
	return time.Time{}
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return outStr, fmt.Errorf("%s", outStr)
	}
	return string(out), nil
}
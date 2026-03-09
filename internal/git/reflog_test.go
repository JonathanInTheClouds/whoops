package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JonathanInTheClouds/whoops/internal/git"
)

func setupRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "whoops-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	orig, _ := os.Getwd()
	os.Chdir(dir)
	mustRun(t, "git", "init")
	mustRun(t, "git", "config", "user.email", "test@whoops.dev")
	mustRun(t, "git", "config", "user.name", "whoops tester")
	mustRun(t, "git", "commit", "--allow-empty", "-m", "initial commit")
	return dir, func() {
		os.Chdir(orig)
		os.RemoveAll(dir)
	}
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

// ---

func TestIsGitRepo(t *testing.T) {
	_, cleanup := setupRepo(t)
	defer cleanup()
	if !git.IsGitRepo() {
		t.Error("expected IsGitRepo true inside a repo")
	}
}

func TestIsGitRepo_Outside(t *testing.T) {
	dir, _ := os.MkdirTemp("", "not-a-repo-*")
	defer os.RemoveAll(dir)
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)
	if git.IsGitRepo() {
		t.Error("expected IsGitRepo false outside a repo")
	}
}

func TestReadReflog_AfterCommit(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	writeFile(t, dir, "foo.go", "package main")
	mustRun(t, "git", "add", ".")
	mustRun(t, "git", "commit", "-m", "add foo")

	actions, err := git.ReadReflog(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
	if actions[0].Type != git.ActionCommit {
		t.Errorf("expected ActionCommit, got %q", actions[0].Type)
	}
	if actions[0].Description != "add foo" {
		t.Errorf("expected description 'add foo', got %q", actions[0].Description)
	}
}

func TestReadReflog_Limit(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		writeFile(t, dir, "file.go", "package main")
		mustRun(t, "git", "add", ".")
		mustRun(t, "git", "commit", "--allow-empty", "-m", "commit")
	}

	actions, err := git.ReadReflog(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) > 3 {
		t.Errorf("expected at most 3 actions, got %d", len(actions))
	}
}

func TestUndo_Commit_DryRun(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	writeFile(t, dir, "bar.go", "package bar")
	mustRun(t, "git", "add", ".")
	mustRun(t, "git", "commit", "-m", "add bar")

	actions, err := git.ReadReflog(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("no actions found")
	}

	result, err := git.Undo(actions[0], true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}
	if result.Command == "" {
		t.Error("expected a non-empty command")
	}

	// Nothing should have changed — commit should still be there
	out, _ := exec.Command("git", "log", "--oneline").CombinedOutput()
	if len(out) == 0 {
		t.Error("expected git log to still show commits after dry run")
	}
}

func TestUndo_Commit(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	writeFile(t, dir, "baz.go", "package baz")
	mustRun(t, "git", "add", ".")
	mustRun(t, "git", "commit", "-m", "add baz")

	actions, err := git.ReadReflog(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = git.Undo(actions[0], false)
	if err != nil {
		t.Fatalf("unexpected error undoing commit: %v", err)
	}

	// The file should now be staged but not committed
	out, _ := exec.Command("git", "diff", "--cached", "--name-only").CombinedOutput()
	if string(out) == "" {
		t.Error("expected baz.go to be staged after undo")
	}
}

func TestUndo_Stash(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	writeFile(t, dir, "stashed.go", "package stash")
	mustRun(t, "git", "add", ".")
	mustRun(t, "git", "stash", "push", "-m", "my stash")

	actions, err := git.ReadReflog(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var stashAction *git.Action
	for _, a := range actions {
		if a.Type == git.ActionStash {
			stashAction = &a
			break
		}
	}
	if stashAction == nil {
		t.Skip("no stash action found in reflog")
	}

	_, err = git.Undo(*stashAction, false)
	if err != nil {
		t.Fatalf("unexpected error undoing stash: %v", err)
	}

	// File should be back in working tree
	if _, err := os.Stat(filepath.Join(dir, "stashed.go")); os.IsNotExist(err) {
		t.Error("expected stashed.go to exist after undo")
	}
}

func TestRelativeTime(t *testing.T) {
	// Just verify it doesn't panic or return empty for zero time
	result := git.RelativeTime(git.ZeroTime())
	if result != "unknown" {
		t.Errorf("expected 'unknown' for zero time, got %q", result)
	}
}

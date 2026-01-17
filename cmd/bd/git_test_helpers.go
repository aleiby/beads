package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/steveyegge/beads/internal/git"
)

// runInDir changes into dir, resets git caches before/after, and executes fn.
// It ensures tests that mutate git repositories don't leak state across cases.
func runInDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	git.ResetCaches()
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
		git.ResetCaches()
	}()
	fn()
}

// runInGitRepo creates a temp dir, initializes git, and runs fn in that directory.
// This eliminates redundant git init calls across tests that need a minimal git repo.
// The git repo has no commits - use setupGitRepo() if you need an initial commit.
func runInGitRepo(t *testing.T, fn func()) {
	t.Helper()
	tmpDir := t.TempDir()
	runInDir(t, tmpDir, func() {
		if err := exec.Command("git", "init", "--initial-branch=main").Run(); err != nil {
			t.Skipf("Skipping test: git init failed: %v", err)
		}
		// Configure git for tests that may need commits
		_ = exec.Command("git", "config", "user.email", "test@test.com").Run()
		_ = exec.Command("git", "config", "user.name", "Test User").Run()
		git.ResetCaches()
		fn()
	})
}

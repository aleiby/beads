package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// filterEnv returns a copy of env with entries matching prefix removed
func filterEnv(env []string, prefix string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			result = append(result, e)
		}
	}
	return result
}

func TestShow_ExternalRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	// Build bd binary
	tmpBin := filepath.Join(t.TempDir(), "bd")
	buildCmd := exec.Command("go", "build", "-o", tmpBin, "./")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build bd: %v\n%s", err, out)
	}

	// Create temp directory for test database
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	dbPath := filepath.Join(beadsDir, "beads.db")
	// Filter all BEADS_ vars and set both BEADS_DIR and BEADS_DB for complete isolation
	testEnv := append(filterEnv(os.Environ(), "BEADS_"),
		"BEADS_DIR="+beadsDir,
		"BEADS_DB="+dbPath,
	)

	// Initialize beads - use isolated env to prevent parent db detection
	initCmd := exec.Command(tmpBin, "init", "--prefix", "test", "--quiet")
	initCmd.Dir = tmpDir
	initCmd.Env = testEnv
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	// Disable contributor routing to ensure issues stay in test database
	// (default routing.mode=auto would route to ~/.beads-planning)
	configCmd := exec.Command(tmpBin, "--no-daemon", "config", "set", "routing.mode", "explicit")
	configCmd.Dir = tmpDir
	configCmd.Env = testEnv
	if out, err := configCmd.CombinedOutput(); err != nil {
		t.Fatalf("config set failed: %v\n%s", err, out)
	}

	// Create issue with external ref - use unique URL to avoid conflicts
	// Include test name in URL to ensure uniqueness across test runs
	externalRef := "https://example.com/test/show_external_ref_" + t.Name() + ".md"
	createCmd := exec.Command(tmpBin, "--no-daemon", "create", "External ref test", "-p", "1",
		"--external-ref", externalRef, "--json")
	createCmd.Dir = tmpDir
	createCmd.Env = testEnv
	createOut, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, createOut)
	}

	var issue map[string]interface{}
	if err := json.Unmarshal(createOut, &issue); err != nil {
		t.Fatalf("failed to parse create output: %v, output: %s", err, createOut)
	}
	id := issue["id"].(string)

	// Show the issue and verify external ref is displayed
	showCmd := exec.Command(tmpBin, "--no-daemon", "show", id)
	showCmd.Dir = tmpDir
	showCmd.Env = testEnv
	showOut, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, showOut)
	}

	out := string(showOut)
	if !strings.Contains(out, "External:") {
		t.Errorf("expected 'External:' in output, got: %s", out)
	}
	if !strings.Contains(out, externalRef) {
		t.Errorf("expected external ref URL %q in output, got: %s", externalRef, out)
	}
}

func TestShow_NoExternalRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	// Build bd binary
	tmpBin := filepath.Join(t.TempDir(), "bd")
	buildCmd := exec.Command("go", "build", "-o", tmpBin, "./")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build bd: %v\n%s", err, out)
	}

	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	dbPath := filepath.Join(beadsDir, "beads.db")
	// Filter all BEADS_ vars and set both BEADS_DIR and BEADS_DB for complete isolation
	testEnv := append(filterEnv(os.Environ(), "BEADS_"),
		"BEADS_DIR="+beadsDir,
		"BEADS_DB="+dbPath,
	)

	// Initialize beads - use isolated env to prevent parent db detection
	initCmd := exec.Command(tmpBin, "init", "--prefix", "test", "--quiet")
	initCmd.Dir = tmpDir
	initCmd.Env = testEnv
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	// Disable contributor routing to ensure issues stay in test database
	// (default routing.mode=auto would route to ~/.beads-planning)
	configCmd := exec.Command(tmpBin, "--no-daemon", "config", "set", "routing.mode", "explicit")
	configCmd.Dir = tmpDir
	configCmd.Env = testEnv
	if out, err := configCmd.CombinedOutput(); err != nil {
		t.Fatalf("config set failed: %v\n%s", err, out)
	}

	// Create issue WITHOUT external ref - use BEADS_DIR to ensure test isolation
	createCmd := exec.Command(tmpBin, "--no-daemon", "create", "No ref test", "-p", "1", "--json")
	createCmd.Dir = tmpDir
	createCmd.Env = testEnv
	createOut, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, createOut)
	}

	var issue map[string]interface{}
	if err := json.Unmarshal(createOut, &issue); err != nil {
		t.Fatalf("failed to parse create output: %v, output: %s", err, createOut)
	}
	id := issue["id"].(string)

	// Show the issue - should NOT contain External Ref line
	showCmd := exec.Command(tmpBin, "--no-daemon", "show", id)
	showCmd.Dir = tmpDir
	showCmd.Env = testEnv
	showOut, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, showOut)
	}

	out := string(showOut)
	if strings.Contains(out, "External:") {
		t.Errorf("expected no 'External:' line for issue without external ref, got: %s", out)
	}
}

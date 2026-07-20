package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestRootHelpUsesStableGroupedLayout(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, text := range []string{"Elyro: Edit on Mac. Build and test in Linux.\n", "Workspace\n", "Project\n", "Agent\n", "Diagnostics\n", "elyro up --open", "elyro exec -- <command>"} {
		if !strings.Contains(got, text) {
			t.Fatalf("help output does not contain %q:\n%s", text, got)
		}
	}
	if strings.Contains(got, "go test ./...") {
		t.Fatalf("help output contains a Toolchain-specific command:\n%s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("buffered help contains ANSI escapes: %q", got)
	}
}

func TestLifecycleHelpDiscoversDryRun(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"up", "down"} {
		cmd := newRootCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{command, "--help"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "--dry-run") {
			t.Fatalf("%s help does not discover --dry-run:\n%s", command, out.String())
		}
	}
}

func TestProcessExitCodePreservesChildExitStatus(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 7").Run()
	if got := processExitCode(fmt.Errorf("remote command failed: %w", err)); got != 7 {
		t.Fatalf("processExitCode() = %d, want 7", got)
	}
}

func TestProcessExitCodeDefaultsToOne(t *testing.T) {
	if got := processExitCode(fmt.Errorf("ordinary error")); got != 1 {
		t.Fatalf("processExitCode() = %d, want 1", got)
	}
}

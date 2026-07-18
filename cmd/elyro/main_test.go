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
	for _, text := range []string{"Elyro\n", "Workspace\n", "Project\n", "Agent integration\n", "Diagnostics\n", "elyro up --open"} {
		if !strings.Contains(got, text) {
			t.Fatalf("help output does not contain %q:\n%s", text, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("buffered help contains ANSI escapes: %q", got)
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

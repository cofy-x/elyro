package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitConfiguresProjectWithoutAgentEnvironments(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	projectDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDir) })
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := runInit(strings.NewReader(""), &output, "", true, false, func(io.Writer) error { return nil }); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "elyro.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "toolchain: go") {
		t.Fatalf("elyro.yaml = %q", data)
	}
	if _, err := os.Stat(filepath.Join(configHome, "elyro", "agent")); !os.IsNotExist(err) {
		t.Fatalf("agent configuration was unexpectedly created: %v", err)
	}
	if !strings.Contains(output.String(), "elyro up") {
		t.Fatalf("runInit output missing next steps:\n%s", output.String())
	}
}

func TestRunInitDoesNotWriteWhenPrerequisitesFail(t *testing.T) {
	projectDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDir) })

	err = runInit(strings.NewReader(""), io.Discard, "go", true, false, func(io.Writer) error {
		return io.ErrUnexpectedEOF
	})
	if err == nil {
		t.Fatal("runInit() error = nil")
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, "elyro.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("elyro.yaml unexpectedly exists: %v", statErr)
	}
}

func TestInitPrerequisiteErrorIncludesRepairSuggestion(t *testing.T) {
	err := initPrerequisiteError(io.ErrUnexpectedEOF, "install the missing tool")
	if err == nil || !strings.Contains(err.Error(), "install the missing tool") {
		t.Fatalf("initPrerequisiteError() = %v", err)
	}
}

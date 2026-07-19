package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProjectWritesDetectedConfigWithYes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := InitProject(InitProjectOptions{ProjectDir: dir, Yes: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "elyro.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); !strings.Contains(got, "default_environment: dev") || !strings.Contains(got, "toolchain: python") {
		t.Fatalf("elyro.yaml = %q", got)
	}
}

func TestInitProjectNonInteractiveRequiresYes(t *testing.T) {
	dir := t.TempDir()
	err := InitProject(InitProjectOptions{ProjectDir: dir, Toolchain: "go", Interactive: false})
	if err == nil || !strings.Contains(err.Error(), "pass --yes") {
		t.Fatalf("InitProject() error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "elyro.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("elyro.yaml unexpectedly exists: %v", statErr)
	}
}

func TestInitProjectDoesNotOverwriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "elyro.yaml")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := InitProject(InitProjectOptions{ProjectDir: dir, Toolchain: "go", Yes: true})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("InitProject() error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "existing\n" {
		t.Fatalf("existing config changed: %q", data)
	}
}

func TestInitProjectReportsExistingConfigBeforeToolchainDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "elyro.yaml")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := InitProject(InitProjectOptions{ProjectDir: dir, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("InitProject() error = %v", err)
	}
}

func TestInitProjectInteractiveSelectionAndConfirmation(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := InitProject(InitProjectOptions{
		ProjectDir:  dir,
		Interactive: true,
		In:          strings.NewReader("2\ny\n"),
		Out:         &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "elyro.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "toolchain: go") {
		t.Fatalf("elyro.yaml = %q", data)
	}
	for _, text := range []string{
		"! No project language was detected\n",
		"? Choose a Toolchain\n",
		"  1  Python\n  2  Go\n  3  Java\n  4  Node.js\n",
		"› Select: ",
		"› Create elyro.yaml with Toolchain Go? [y/N] ",
		"✓ Created elyro.yaml\n",
	} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("init output missing %q:\n%s", text, out.String())
		}
	}
}

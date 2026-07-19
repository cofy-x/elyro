package cliui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPlainRenderer(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewForTest(&out, false)
	if err := ui.Brand("Elyro", "Edit on Mac. Build and test in Linux."); err != nil {
		t.Fatal(err)
	}
	if err := ui.Section("Workspace"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Question("Choose a Toolchain"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Prompt("Select: "); err != nil {
		t.Fatal(err)
	}
	if err := ui.Text("2"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Success("Workspace ready"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Fields(Field{"project", "/tmp/demo"}, Field{"toolchain", "go"}); err != nil {
		t.Fatal(err)
	}
	if err := ui.Next("elyro shell", "elyro exec -- go test ./..."); err != nil {
		t.Fatal(err)
	}
	want := "Elyro: Edit on Mac. Build and test in Linux.\nWorkspace\n? Choose a Toolchain\n› Select: 2\n✓ Workspace ready\n  project    /tmp/demo\n  toolchain  go\n\nNext\n  elyro shell\n  elyro exec -- go test ./...\n"
	if got := out.String(); got != want {
		t.Fatalf("plain output = %q, want %q", got, want)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatal("plain output contains ANSI escapes")
	}
}

func TestColorRenderer(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewForTest(&out, true)
	if err := ui.Brand("Elyro", "Edit on Mac. Build and test in Linux."); err != nil {
		t.Fatal(err)
	}
	if err := ui.Section("Workspace"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Question("Choose an editor"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Progress("Preparing Workspace"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Success("Workspace ready"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Warning("No project language was detected"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Failure("Docker is unavailable"); err != nil {
		t.Fatal(err)
	}
	for role, sequence := range map[string]string{
		"brand":    "\x1b[1;94mElyro:\x1b[m",
		"section":  "\x1b[1;34mWorkspace\x1b[m",
		"question": "\x1b[1;34m? Choose an editor\x1b[m",
		"progress": "\x1b[36m→ Preparing Workspace\x1b[m",
		"success":  "\x1b[1;32m✓ Workspace ready\x1b[m",
		"warning":  "\x1b[33m! No project language was detected\x1b[m",
		"failure":  "\x1b[1;31m✗ Docker is unavailable\x1b[m",
	} {
		if !strings.Contains(out.String(), sequence) {
			t.Fatalf("colored output missing %s sequence %q: %q", role, sequence, out.String())
		}
	}
}

func TestColoredFieldsKeepSeparatorAfterLongestLabel(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewForTest(&out, true)
	if err := ui.Fields(Field{"short", "one"}, Field{"privileged", "false"}); err != nil {
		t.Fatal(err)
	}

	plain := stripANSI(out.String())
	if want := "  short       one\n  privileged  false\n"; plain != want {
		t.Fatalf("colored fields = %q, want %q", plain, want)
	}
}

func TestColorDisabledByEnvironmentAndWriter(t *testing.T) {
	t.Parallel()

	for _, env := range [][]string{{"NO_COLOR=1"}, {"TERM=dumb"}, {"CI=true"}} {
		if colorEnabled(os.Stdout, env) {
			t.Fatalf("color enabled for env %v", env)
		}
	}
	if colorEnabled(&bytes.Buffer{}, nil) {
		t.Fatal("color enabled for non-file writer")
	}
}

func stripANSI(value string) string {
	for {
		start := strings.Index(value, "\x1b[")
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start:], 'm')
		if end < 0 {
			return value
		}
		value = value[:start] + value[start+end+1:]
	}
}

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
	if err := ui.Success("Workspace ready"); err != nil {
		t.Fatal(err)
	}
	if err := ui.Fields(Field{"project", "/tmp/demo"}, Field{"toolchain", "go"}); err != nil {
		t.Fatal(err)
	}
	if err := ui.Next("elyro shell", "elyro exec -- go test ./..."); err != nil {
		t.Fatal(err)
	}
	want := "✓ Workspace ready\n  project    /tmp/demo\n  toolchain  go\n\nNext\n  elyro shell\n  elyro exec -- go test ./...\n"
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
	if err := ui.Failure("Docker is unavailable"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[") || !strings.Contains(out.String(), "Docker is unavailable") {
		t.Fatalf("colored output = %q", out.String())
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

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace/editor"
	"github.com/spf13/cobra"
)

func TestRootRejectsUnknownCommand(t *testing.T) {
	cmd := &cobra.Command{Use: "elyro"}
	cmd.AddCommand(NewWorkspaceCommands()...)
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"missing-command"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Execute(missing-command) error = %v, want unknown command", err)
	}
}

func TestFindElyroRepoRootUsesImagesLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("test:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	imageDir := filepath.Join(root, "images", "workspace-base")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "apps", "workspace")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := findElyroRepoRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("findElyroRepoRoot = %q, want %q", got, root)
	}
}

func TestPromptEditorSelectionDefaultSkip(t *testing.T) {
	t.Parallel()

	options := []editor.Option{{Label: "Cursor"}, {Label: "VS Code"}}
	var out bytes.Buffer
	got := promptEditorSelection(strings.NewReader("\n"), &out, options)
	if got != -1 {
		t.Fatalf("promptEditorSelection = %d, want -1", got)
	}
}

func TestPromptEditorSelectionValidChoice(t *testing.T) {
	t.Parallel()

	options := []editor.Option{{Label: "Cursor"}, {Label: "VS Code"}}
	var out bytes.Buffer
	got := promptEditorSelection(strings.NewReader("2\n"), &out, options)
	if got != 1 {
		t.Fatalf("promptEditorSelection = %d, want 1", got)
	}
}

func TestPromptEditorSelectionRetryInvalidChoice(t *testing.T) {
	t.Parallel()

	options := []editor.Option{{Label: "Cursor"}, {Label: "VS Code"}}
	var out bytes.Buffer
	got := promptEditorSelection(strings.NewReader("9\n1\n"), &out, options)
	if got != 0 {
		t.Fatalf("promptEditorSelection = %d, want 0", got)
	}
	if !strings.Contains(out.String(), "Invalid selection.") {
		t.Fatalf("expected invalid selection prompt, got %q", out.String())
	}
}

func TestDisplayOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: "none"},
		{name: "space", value: "  ", want: "none"},
		{name: "value", value: "18080:8000", want: "18080:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := displayOptional(tt.value)
			if got != tt.want {
				t.Fatalf("displayOptional(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

package cli

import (
	"strings"
	"testing"
)

func TestUpEditorRequiresOpen(t *testing.T) {
	cmd := newUpCmd(&GlobalOptions{ProjectDir: ".", SSHConfigPath: "~/.ssh/config"})
	cmd.SetArgs([]string{"--editor", "cursor"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--editor requires --open") {
		t.Fatalf("up --editor error = %v, want --open requirement", err)
	}
}

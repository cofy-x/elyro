package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cofy-x/elyro/skills"
)

func TestSkillInstallIsIdempotentAndProtectsModifiedContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewSkillCmd()
	cmd.SetArgs([]string{"install", "codex"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cmd = NewSkillCmd()
	cmd.SetArgs([]string{"install", "codex"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("idempotent install failed: %v", err)
	}

	skillFile := filepath.Join(home, ".agents", "skills", elyroSkillName, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = NewSkillCmd()
	cmd.SetArgs([]string{"install", "codex"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("install error = %v, want different-content refusal", err)
	}
	cmd = NewSkillCmd()
	cmd.SetArgs([]string{"install", "codex", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if match, exists, err := skillMatches(filepath.Dir(skillFile)); err != nil || !exists || !match {
		t.Fatalf("forced install match = %t, exists = %t, error = %v", match, exists, err)
	}
	if err := os.WriteFile(skillFile, []byte("modified again\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = NewSkillCmd()
	cmd.SetArgs([]string{"uninstall", "codex"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatalf("uninstall error = %v, want modified-content refusal", err)
	}
	cmd = NewSkillCmd()
	cmd.SetArgs([]string{"uninstall", "codex", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(skillFile)); !os.IsNotExist(err) {
		t.Fatalf("skill directory still exists: %v", err)
	}
}

func TestSkillInstallAllPreflightsConflicts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := filepath.Join(home, ".claude", "skills", elyroSkillName)
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "SKILL.md"), []byte("user content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewSkillCmd()
	cmd.SetArgs([]string{"install", "all"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("install all succeeded despite conflict")
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", elyroSkillName)); !os.IsNotExist(err) {
		t.Fatalf("Codex skill was partially installed: %v", err)
	}
}

func TestSkillShowPrintsEmbeddedSource(t *testing.T) {
	cmd := NewSkillCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"show"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), skills.SkillMarkdown) {
		t.Fatalf("skill show changed embedded bytes")
	}
}

func TestSkillHelpDiscoversInspectionAndInstall(t *testing.T) {
	cmd := NewSkillCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"Inspect or install the Elyro Skill for coding agents",
		"Print the complete embedded Elyro Skill",
		"Install the embedded Elyro Skill for a host coding agent",
		"Uninstall the embedded Elyro Skill",
		"elyro skill show",
		"elyro skill install codex",
	} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("skill help does not contain %q:\n%s", text, out.String())
		}
	}
}

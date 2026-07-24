package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/skills"
	"github.com/spf13/cobra"
)

const elyroSkillName = "use-elyro-workspace"

type skillTarget struct {
	name string
	dir  string
}

func NewSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Inspect or install the Elyro Skill for coding agents",
		Args:  cobra.NoArgs,
		Example: `  elyro skill show
  elyro skill install codex`,
	}
	cmd.AddCommand(newSkillShowCmd(), newSkillInstallCmd(), newSkillUninstallCmd())
	return cmd
}

func newSkillShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the complete embedded Elyro Skill",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write(skills.SkillMarkdown)
			return err
		},
	}
}

func newSkillInstallCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "install <codex|claude-code|all>",
		Short: "Install the embedded Elyro Skill for a host coding agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ui := cliui.New(cmd.OutOrStdout())
			targets, err := resolveSkillTargets(args[0])
			if err != nil {
				return err
			}
			for _, target := range targets {
				match, exists, err := skillMatches(target.dir)
				if err != nil {
					return err
				}
				if exists && !match && !force {
					return fmt.Errorf("%s skill already exists with different content at %s; use --force to replace it", target.name, target.dir)
				}
			}
			for _, target := range targets {
				if err := installSkill(target.dir, force); err != nil {
					return fmt.Errorf("install %s skill: %w", target.name, err)
				}
				if err := ui.Success(fmt.Sprintf("Installed %s skill", target.name)); err != nil {
					return err
				}
				if err := ui.Fields(cliui.Field{Label: "path", Value: target.dir}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Replace an existing skill with different content")
	return cmd
}

func newSkillUninstallCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "uninstall <codex|claude-code|all>",
		Short: "Uninstall the embedded Elyro Skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ui := cliui.New(cmd.OutOrStdout())
			targets, err := resolveSkillTargets(args[0])
			if err != nil {
				return err
			}
			for _, target := range targets {
				match, exists, err := skillMatches(target.dir)
				if err != nil {
					return err
				}
				if exists && !match && !force {
					return fmt.Errorf("%s skill contains modified content at %s; use --force to remove it", target.name, target.dir)
				}
			}
			for _, target := range targets {
				if err := os.RemoveAll(target.dir); err != nil {
					return fmt.Errorf("uninstall %s skill: %w", target.name, err)
				}
				if err := ui.Success(fmt.Sprintf("Uninstalled %s skill", target.name)); err != nil {
					return err
				}
				if err := ui.Fields(cliui.Field{Label: "path", Value: target.dir}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove a skill even when its content was modified")
	return cmd
}

func resolveSkillTargets(selection string) ([]skillTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	all := map[string]skillTarget{
		"codex":       {name: "Codex", dir: filepath.Join(home, ".agents", "skills", elyroSkillName)},
		"claude-code": {name: "Claude Code", dir: filepath.Join(home, ".claude", "skills", elyroSkillName)},
	}
	if selection == "all" {
		return []skillTarget{all["codex"], all["claude-code"]}, nil
	}
	target, ok := all[selection]
	if !ok {
		return nil, fmt.Errorf("unsupported skill target %q (supported: codex, claude-code, all)", selection)
	}
	return []skillTarget{target}, nil
}

func embeddedSkillFiles() map[string][]byte {
	return map[string][]byte{
		"SKILL.md":           skills.SkillMarkdown,
		"agents/openai.yaml": skills.OpenAIYAML,
	}
}

func skillMatches(dir string) (bool, bool, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, true, nil
	}
	expected := embeddedSkillFiles()
	var actual []string
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("skill directory contains a symbolic link")
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		actual = append(actual, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return false, true, err
	}
	sort.Strings(actual)
	want := make([]string, 0, len(expected))
	for name := range expected {
		want = append(want, name)
	}
	sort.Strings(want)
	if strings.Join(actual, "\x00") != strings.Join(want, "\x00") {
		return false, true, nil
	}
	for name, content := range expected {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			return false, true, err
		}
		if !bytes.Equal(data, content) {
			return false, true, nil
		}
	}
	return true, true, nil
}

func installSkill(dir string, force bool) error {
	match, exists, err := skillMatches(dir)
	if err != nil {
		return err
	}
	if match {
		return nil
	}
	if exists && !force {
		return errors.New("existing skill content differs")
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(parent, ".elyro-skill-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	for name, content := range embeddedSkillFiles() {
		path := filepath.Join(temp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	if exists {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return os.Rename(temp, dir)
}

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/cofy-x/elyro/internal/cliui"
	workspacecli "github.com/cofy-x/elyro/internal/workspace/cli"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		ui := cliui.New(os.Stderr)
		if ui.ColorEnabled() {
			_ = ui.Failure(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(processExitCode(err))
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "elyro",
		Short:         "Edit on Mac. Build and test in Linux.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpFunc(printCommandHelp)

	rootCmd.AddCommand(
		newDoctorCmd(),
		newInitCmd(),
		newVersionCmd(),
		workspacecli.NewSkillCmd(),
	)
	rootCmd.AddCommand(workspacecli.NewWorkspaceCommands()...)
	return rootCmd
}

func printCommandHelp(cmd *cobra.Command, _ []string) {
	ui := cliui.New(cmd.OutOrStdout())
	if cmd.Name() != "elyro" {
		_ = ui.Section(cmd.CommandPath())
		_ = ui.Text(cmd.Short)
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
		return
	}

	_ = ui.Brand("Elyro", "Edit on Mac. Build and test in Linux.")
	_ = ui.Next("elyro up --open", "elyro exec -- <command>")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	for _, group := range []struct {
		title    string
		commands []string
	}{
		{"Workspace", []string{"up", "down", "shell", "exec", "open", "status", "list"}},
		{"Project", []string{"init"}},
		{"Agent", []string{"skill"}},
		{"Diagnostics", []string{"doctor", "version", "help"}},
	} {
		_ = ui.Section(group.title)
		fields := make([]cliui.Field, 0, len(group.commands))
		for _, name := range group.commands {
			child, _, err := cmd.Find([]string{name})
			if err != nil || child == nil {
				continue
			}
			fields = append(fields, cliui.Field{Label: name, Value: child.Short})
		}
		_ = ui.Fields(fields...)
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}
}

func processExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code > 0 {
			return code
		}
	}
	return 1
}

package cli

import "github.com/spf13/cobra"

type GlobalOptions struct {
	ProjectDir    string
	SSHConfigPath string
}

func NewWorkspaceCommands() []*cobra.Command {
	opts := &GlobalOptions{ProjectDir: ".", SSHConfigPath: "~/.ssh/config"}
	commands := []*cobra.Command{
		newUpCmd(opts),
		newListCmd(opts),
		newShellCmd(opts),
		newExecCmd(opts),
		newOpenCmd(opts),
		newDownCmd(opts),
		newStatusCmd(opts),
	}
	for _, command := range commands {
		command.Flags().StringVar(&opts.ProjectDir, "project-dir", ".", "Project directory to mount into the workspace")
	}
	return commands
}

package cli

import (
	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/spf13/cobra"
)

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

func resolvedProjectDir(cmd *cobra.Command, opts *GlobalOptions) (string, error) {
	root, err := workspace.ResolveProjectRoot(opts.ProjectDir, cmd.Flags().Changed("project-dir"))
	if err != nil {
		return "", err
	}
	return root.Dir, nil
}

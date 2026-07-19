package main

import (
	"io"

	"github.com/cofy-x/elyro/internal/workspace"
	workspacecli "github.com/cofy-x/elyro/internal/workspace/cli"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var toolchain string
	var yes bool
	var projectDir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create elyro.yaml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in := cmd.InOrStdin()
			out := cmd.OutOrStdout()
			root, err := workspace.ResolveProjectRoot(projectDir, cmd.Flags().Changed("project-dir"))
			if err != nil {
				return err
			}
			return runInitAt(
				in,
				out,
				root.Dir,
				toolchain,
				yes,
				isTerminalFile(stdinFile(in)) && isTerminalFile(stdoutFile(out)),
			)
		},
	}
	cmd.Flags().StringVar(&toolchain, "toolchain", "", "Workspace toolchain (python, go, java, or node); detected when omitted")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Create elyro.yaml without prompting")
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory to configure")
	return cmd
}

func runInit(in io.Reader, out io.Writer, toolchain string, yes, interactive bool) error {
	return runInitAt(in, out, ".", toolchain, yes, interactive)
}

func runInitAt(in io.Reader, out io.Writer, projectDir, toolchain string, yes, interactive bool) error {
	return workspacecli.InitProject(workspacecli.InitProjectOptions{
		ProjectDir:  projectDir,
		Toolchain:   toolchain,
		Yes:         yes,
		In:          in,
		Out:         out,
		Interactive: interactive,
	})
}

package main

import (
	"fmt"
	"io"

	workspacecli "github.com/cofy-x/elyro/internal/workspace/cli"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var toolchain string
	var yes bool
	var projectDir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create elyro.yaml for the current project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in := cmd.InOrStdin()
			out := cmd.OutOrStdout()
			return runInitAt(
				in,
				out,
				projectDir,
				toolchain,
				yes,
				isTerminalFile(stdinFile(in)) && isTerminalFile(stdoutFile(out)),
				runInitPrerequisites,
			)
		},
	}
	cmd.Flags().StringVar(&toolchain, "toolchain", "", "Workspace toolchain (python, go, java, or node); detected when omitted")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Create elyro.yaml without prompting")
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory to configure")
	return cmd
}

func runInit(in io.Reader, out io.Writer, toolchain string, yes, interactive bool, check func(io.Writer) error) error {
	return runInitAt(in, out, ".", toolchain, yes, interactive, check)
}

func runInitAt(in io.Reader, out io.Writer, projectDir, toolchain string, yes, interactive bool, check func(io.Writer) error) error {
	if err := check(out); err != nil {
		return err
	}
	return workspacecli.InitProject(workspacecli.InitProjectOptions{
		ProjectDir:  projectDir,
		Toolchain:   toolchain,
		Yes:         yes,
		In:          in,
		Out:         out,
		Interactive: interactive,
	})
}

func runInitPrerequisites(out io.Writer) error {
	dockerErr := checkCommand("docker")
	sshErr := checkCommand("ssh")
	dockerDaemonErr := fmt.Errorf("not checked because the Docker CLI is unavailable")
	if dockerErr == nil {
		dockerDaemonErr = checkDockerDaemon()
	}
	checks := []doctorCheck{
		{name: "docker", required: true, err: initPrerequisiteError(dockerErr, "install Docker and ensure `docker` is in PATH")},
		{name: "ssh", required: true, err: initPrerequisiteError(sshErr, "install an OpenSSH client and ensure `ssh` is in PATH")},
		{name: "docker daemon", required: true, err: initPrerequisiteError(dockerDaemonErr, "start Docker and verify that `docker info` succeeds")},
	}
	return printDoctorChecks(out, "Elyro init prerequisites:", checks)
}

func initPrerequisiteError(err error, suggestion string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%v; %s", err, suggestion)
}

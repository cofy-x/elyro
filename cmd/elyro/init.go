package main

import (
	"fmt"
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
		Short: "Create elyro.yaml for the current project",
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
	report := doctorJSONView{SchemaVersion: 2, Kind: "doctor", Healthy: true}
	report.add(initPrerequisiteCheck("docker_cli", "Docker CLI is available", dockerErr, "install Docker and ensure `docker` is in PATH"))
	report.add(initPrerequisiteCheck("ssh_cli", "OpenSSH client is available", sshErr, "install an OpenSSH client and ensure `ssh` is in PATH"))
	report.add(initPrerequisiteCheck("docker_daemon", "Docker daemon is reachable", dockerDaemonErr, "start Docker and verify that `docker info` succeeds"))
	if err := printDoctorReport(out, report); err != nil {
		return err
	}
	if !report.Healthy {
		return fmt.Errorf("one or more checks failed")
	}
	return nil
}

func initPrerequisiteCheck(name, success string, err error, suggestion string) doctorCheck {
	if err == nil {
		return doctorCheck{Scope: "system", Name: name, Status: doctorStatusOK, Message: success}
	}
	return doctorCheck{Scope: "system", Name: name, Status: doctorStatusFail, Message: fmt.Sprintf("%v; %s", err, suggestion)}
}

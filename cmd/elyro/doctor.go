package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroversion "github.com/cofy-x/elyro/internal/version"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/access"
	"github.com/cofy-x/elyro/internal/workspace/editor"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
	"github.com/spf13/cobra"
)

const (
	doctorStatusOK   doctorStatus = "ok"
	doctorStatusWarn doctorStatus = "warn"
	doctorStatusFail doctorStatus = "fail"
)

type doctorStatus string

type doctorCheck struct {
	Scope   string       `json:"scope"`
	Name    string       `json:"name"`
	Status  doctorStatus `json:"status"`
	Message string       `json:"message"`
}

type doctorProjectView struct {
	Root            string                           `json:"root"`
	Source          elyroworkspace.ProjectRootSource `json:"source"`
	ConfigPath      string                           `json:"config_path,omitempty"`
	Environment     string                           `json:"environment,omitempty"`
	Toolchain       string                           `json:"toolchain,omitempty"`
	Image           string                           `json:"image,omitempty"`
	Platform        string                           `json:"platform,omitempty"`
	WorkspaceStatus string                           `json:"workspace_status,omitempty"`
}

type doctorJSONView struct {
	SchemaVersion int                `json:"schema_version"`
	Kind          string             `json:"kind"`
	Healthy       bool               `json:"healthy"`
	Project       *doctorProjectView `json:"project,omitempty"`
	Checks        []doctorCheck      `json:"checks"`
}

func newDoctorCmd() *cobra.Command {
	var outputJSON bool
	var projectDir string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check Elyro system and project health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := doctorReport(projectDir, cmd.Flags().Changed("project-dir"))
			if outputJSON {
				if err := writeDoctorJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else if err := printDoctorReport(cmd.OutOrStdout(), report); err != nil {
				return err
			}
			if !report.Healthy {
				return errors.New("one or more checks failed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print checks as JSON")
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory to diagnose")
	return cmd
}

func doctorReport(projectDir string, explicit bool) doctorJSONView {
	report := doctorJSONView{SchemaVersion: 2, Kind: "doctor", Healthy: true, Checks: []doctorCheck{}}
	dockerPath, dockerErr := exec.LookPath("docker")
	report.add(commandDoctorCheck("system", "docker_cli", "Docker CLI", dockerPath, dockerErr))
	sshPath, sshErr := exec.LookPath("ssh")
	report.add(commandDoctorCheck("system", "ssh_cli", "OpenSSH client", sshPath, sshErr))
	if !elyroversion.IsRelease() {
		goPath, goErr := exec.LookPath("go")
		report.add(commandDoctorCheck("system", "go_cli", "Go development toolchain", goPath, goErr))
	}
	daemonOK := dockerErr == nil
	if dockerErr != nil {
		report.add(doctorCheck{Scope: "system", Name: "docker_daemon", Status: doctorStatusFail, Message: "Docker daemon was not checked because the Docker CLI is unavailable"})
	} else if err := checkDockerDaemon(); err != nil {
		daemonOK = false
		report.add(doctorCheck{Scope: "system", Name: "docker_daemon", Status: doctorStatusFail, Message: err.Error()})
	} else {
		report.add(doctorCheck{Scope: "system", Name: "docker_daemon", Status: doctorStatusOK, Message: "Docker daemon is reachable"})
	}

	store, storePath, storeErr := loadDoctorRegistry()
	if storeErr != nil {
		report.add(doctorCheck{Scope: "system", Name: "workspace_registry", Status: doctorStatusFail, Message: storeErr.Error()})
	} else {
		report.add(doctorCheck{Scope: "system", Name: "workspace_registry", Status: doctorStatusOK, Message: fmt.Sprintf("Workspace registry is readable at %s", storePath)})
	}

	projectMode := explicit
	if !projectMode {
		if detected, err := elyroworkspace.HasProjectSignal(projectDir); err == nil {
			projectMode = detected
		} else if storeErr == nil {
			report.add(doctorCheck{Scope: "project", Name: "project_detection", Status: doctorStatusFail, Message: err.Error()})
			projectMode = true
		}
	}
	if projectMode {
		addProjectDoctorChecks(&report, projectDir, explicit, store, storeErr, daemonOK, sshErr == nil)
	}
	return report
}

func addProjectDoctorChecks(report *doctorJSONView, projectDir string, explicit bool, store elyroworkspace.Store, storeErr error, daemonOK, sshOK bool) {
	root, err := elyroworkspace.ResolveProjectRoot(projectDir, explicit)
	if err != nil {
		report.add(doctorCheck{Scope: "project", Name: "project_root", Status: doctorStatusFail, Message: err.Error()})
		return
	}
	view := &doctorProjectView{Root: root.Dir, Source: root.Source, ConfigPath: root.ConfigPath}
	report.Project = view
	report.add(doctorCheck{Scope: "project", Name: "project_root", Status: doctorStatusOK, Message: fmt.Sprintf("Resolved %s project root %s", root.Source, root.Dir)})

	project, err := elyroworkspace.ResolveProjectContext(root.Dir, "", "")
	if err != nil {
		report.add(doctorCheck{Scope: "project", Name: "workspace_identity", Status: doctorStatusFail, Message: err.Error()})
		return
	}
	environment, err := elyroworkspace.ResolveEnvironment(root.Dir, project.MountDir, elyroworkspace.EnvironmentSelection{Platform: elyroworkspace.DefaultPlatform})
	if err != nil {
		if projectConfigurationExists(root.Dir) {
			report.add(doctorCheck{Scope: "project", Name: "workspace_configuration", Status: doctorStatusFail, Message: err.Error()})
			return
		}
		report.add(doctorCheck{Scope: "project", Name: "workspace_configuration", Status: doctorStatusWarn, Message: fmt.Sprintf("No Workspace Environment is selected: %v; `elyro up` can use an explicit --toolchain", err)})
		addUnresolvedWorkspaceDoctorChecks(report, root.Dir, project.MountDir, daemonOK)
		return
	}
	if root.ConfigPath == "" && environment.ProjectConfigured {
		view.ConfigPath = filepath.Join(root.Dir, "elyro.yaml")
	}
	view.Environment = environment.Name
	view.Toolchain = string(environment.Toolchain)
	view.Image = environment.Image
	view.Platform = environment.Platform
	report.add(doctorCheck{Scope: "project", Name: "workspace_configuration", Status: doctorStatusOK, Message: describeEnvironment(environment)})

	if !daemonOK {
		report.add(doctorCheck{Scope: "project", Name: "workspace_image", Status: doctorStatusWarn, Message: "Workspace image was not checked because the Docker daemon is unavailable"})
	} else if dockerruntime.ImageExists(context.Background(), environment.Image) {
		report.add(doctorCheck{Scope: "project", Name: "workspace_image", Status: doctorStatusOK, Message: fmt.Sprintf("Image is available: %s", environment.Image)})
	} else if environment.CustomImage {
		report.add(doctorCheck{Scope: "project", Name: "workspace_image", Status: doctorStatusFail, Message: fmt.Sprintf("Custom image is missing: %s; build or pull it before running `elyro up`", environment.Image)})
	} else {
		report.add(doctorCheck{Scope: "project", Name: "workspace_image", Status: doctorStatusWarn, Message: fmt.Sprintf("Official image is not local and will be pulled by `elyro up`: %s", environment.Image)})
	}

	if !daemonOK {
		return
	}
	status, err := local.Status(context.Background(), local.StatusRequest{ProjectDir: root.Dir})
	if err != nil {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusFail, Message: err.Error()})
		return
	}
	if status.Container == nil {
		view.WorkspaceStatus = "absent"
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusWarn, Message: "Workspace is absent; run `elyro up` when it is needed"})
		addEditorDoctorCheck(report, "", project.MountDir)
		return
	}
	view.WorkspaceStatus = status.Container.Status
	if status.Container.Status == "running" {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusOK, Message: "Workspace is running"})
	} else {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusWarn, Message: fmt.Sprintf("Workspace is %s; `elyro up` will start it", status.Container.Status)})
	}
	if local.ContainerSpecificationMatches(
		status.Container,
		project,
		environment,
		elyroworkspace.NormalizePublishSpecs(environment.Docker.Publishes),
		elyroworkspace.NormalizeDockerMounts(environment.Docker.Mounts),
		fmt.Sprintf("%t", environment.Docker.Privileged),
	) {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_specification", Status: doctorStatusOK, Message: "Workspace matches the resolved project configuration"})
	} else {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_specification", Status: doctorStatusWarn, Message: "Workspace differs from the resolved project configuration; `elyro up` will recreate it"})
	}

	if storeErr == nil {
		record, recordErr := elyroworkspace.Current(store, root.Dir)
		if recordErr != nil || record.ProjectDir != root.Dir {
			report.add(doctorCheck{Scope: "workspace", Name: "workspace_registration", Status: doctorStatusFail, Message: "Running Workspace is missing its exact registry record; run `elyro up --recreate`"})
		} else {
			report.add(doctorCheck{Scope: "workspace", Name: "workspace_registration", Status: doctorStatusOK, Message: "Workspace registry record is present"})
			if sshOK {
				sshConfig, expandErr := elyroworkspace.ExpandPath("~/.ssh/config")
				if expandErr != nil {
					report.add(doctorCheck{Scope: "workspace", Name: "managed_ssh", Status: doctorStatusFail, Message: expandErr.Error()})
				} else {
					identityFile, identityErr := doctorIdentityFile()
					entry := elyroworkspace.SSHHostEntry{
						HostAlias: record.SSHAlias, HostName: "127.0.0.1", Port: status.Container.HostPort, User: "elyro",
						IdentityFile: identityFile, KnownHostsFile: elyroworkspace.DefaultKnownHostsFile(),
					}
					validateErr := identityErr
					if validateErr == nil {
						_, validateErr = os.Stat(identityFile)
					}
					if validateErr == nil {
						validateErr = elyroworkspace.ValidateManagedSSHHostEntry(sshConfig, entry)
					}
					if validateErr == nil {
						validateErr = elyroworkspace.ValidateKnownSSHHost(entry.KnownHostsFile, record.SSHAlias, status.Container.ID)
					}
					if validateErr != nil {
						report.add(doctorCheck{Scope: "workspace", Name: "managed_ssh", Status: doctorStatusFail, Message: validateErr.Error()})
					} else {
						report.add(doctorCheck{Scope: "workspace", Name: "managed_ssh", Status: doctorStatusOK, Message: "Managed SSH configuration, identity, and known-host trust match the running Workspace"})
					}
				}
			}
			addEditorDoctorCheck(report, record.SSHAlias, record.ContainerWorkspaceDir)
		}
	}
}

func projectConfigurationExists(projectDir string) bool {
	_, err := os.Lstat(filepath.Join(projectDir, "elyro.yaml"))
	return err == nil || !os.IsNotExist(err)
}

func addUnresolvedWorkspaceDoctorChecks(report *doctorJSONView, projectDir, mountDir string, daemonOK bool) {
	if !daemonOK {
		return
	}
	status, err := local.Status(context.Background(), local.StatusRequest{ProjectDir: projectDir})
	if err != nil {
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusFail, Message: err.Error()})
		return
	}
	if status.Container == nil {
		report.Project.WorkspaceStatus = "absent"
		report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: doctorStatusWarn, Message: "Workspace is absent; choose a Toolchain with `elyro up --toolchain` when it is needed"})
		addEditorDoctorCheck(report, "", mountDir)
		return
	}
	report.Project.WorkspaceStatus = status.Container.Status
	statusLevel := doctorStatusWarn
	message := fmt.Sprintf("Workspace is %s, but no Environment can be resolved for specification checks", status.Container.Status)
	if status.Container.Status == "running" {
		statusLevel = doctorStatusOK
		message = "Workspace is running, but no Environment can be resolved for specification checks"
	}
	report.add(doctorCheck{Scope: "workspace", Name: "workspace_state", Status: statusLevel, Message: message})
	addEditorDoctorCheck(report, "", mountDir)
}

func addEditorDoctorCheck(report *doctorJSONView, hostAlias, remoteDir string) {
	options := editor.DetectOptions(hostAlias, remoteDir)
	if len(options) == 0 {
		report.add(doctorCheck{Scope: "editor", Name: "supported_editor", Status: doctorStatusWarn, Message: "No Cursor or VS Code command was found in PATH; editor handoff is optional"})
		return
	}
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	report.add(doctorCheck{Scope: "editor", Name: "supported_editor", Status: doctorStatusOK, Message: "Detected " + strings.Join(labels, ", ")})
}

func describeEnvironment(environment elyroworkspace.ResolvedEnvironment) string {
	parts := []string{fmt.Sprintf("environment=%s", environment.Name), fmt.Sprintf("image=%s", environment.Image), fmt.Sprintf("platform=%s", environment.Platform)}
	if environment.Toolchain != "" {
		parts = append(parts, fmt.Sprintf("toolchain=%s", environment.Toolchain))
	}
	return "Resolved " + strings.Join(parts, " ")
}

func doctorIdentityFile() (string, error) {
	path, err := elyroworkspace.ExpandPath(access.DefaultWorkspaceIdentityFile)
	if err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

func commandDoctorCheck(scope, name, label, path string, err error) doctorCheck {
	if err != nil {
		return doctorCheck{Scope: scope, Name: name, Status: doctorStatusFail, Message: fmt.Sprintf("%s was not found in PATH", label)}
	}
	return doctorCheck{Scope: scope, Name: name, Status: doctorStatusOK, Message: fmt.Sprintf("%s is available at %s", label, path)}
}

func checkCommand(name string) error {
	_, err := exec.LookPath(name)
	return err
}

func loadDoctorRegistry() (elyroworkspace.Store, string, error) {
	path, err := elyroworkspace.DefaultPath()
	if err != nil {
		return elyroworkspace.Store{}, "", err
	}
	store, err := elyroworkspace.Load(path)
	if err != nil {
		return elyroworkspace.Store{}, path, err
	}
	return store, path, nil
}

func (report *doctorJSONView) add(check doctorCheck) {
	if check.Status == doctorStatusFail {
		report.Healthy = false
	}
	report.Checks = append(report.Checks, check)
}

func writeDoctorJSON(out io.Writer, report doctorJSONView) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func printDoctorReport(out io.Writer, report doctorJSONView) error {
	ui := cliui.New(out)
	labels := map[string]string{"system": "System", "project": "Project", "workspace": "Workspace", "editor": "Editor"}
	for _, scope := range []string{"system", "project", "workspace", "editor"} {
		printed := false
		for _, check := range report.Checks {
			if check.Scope != scope {
				continue
			}
			if !printed {
				if err := ui.Title(labels[scope]); err != nil {
					return err
				}
				printed = true
			}
			message := strings.ReplaceAll(check.Name, "_", " ") + ": " + check.Message
			var err error
			switch check.Status {
			case doctorStatusOK:
				err = ui.Success(message)
			case doctorStatusWarn:
				err = ui.Warning(message)
			default:
				err = ui.Failure(message)
			}
			if err != nil {
				return err
			}
		}
		if printed {
			if err := ui.Text(""); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDockerDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

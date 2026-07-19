package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/access"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

func newUpCmd(opts *GlobalOptions) *cobra.Command {
	var toolchain string
	var environmentName string
	var platform string
	var allowUnsafeEnvironment bool
	var publishSpecs []string
	var openEditor bool
	var editorName string
	var outputJSON bool
	var recreate bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start or reuse a Workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := resolvedProjectDir(cmd, opts)
			if err != nil {
				return err
			}
			stdoutUI := cliui.New(cmd.OutOrStdout())
			stderrUI := cliui.New(cmd.ErrOrStderr())
			if cmd.Flags().Changed("editor") && !openEditor {
				return errors.New("--editor requires --open")
			}
			if outputJSON && openEditor {
				return errors.New("--json cannot be combined with --open")
			}
			sshConfigPath, err := expandSSHConfigPath(opts.SSHConfigPath)
			if err != nil {
				return err
			}

			ctx, cancel := signalContext()
			defer cancel()
			request := local.UpRequest{
				ProjectDir:             projectDir,
				SSHConfigPath:          sshConfigPath,
				IdentityFile:           access.DefaultWorkspaceIdentityFile,
				AllowUnsafeEnvironment: allowUnsafeEnvironment,
				Toolchain:              toolchain,
				Environment:            environmentName,
				Platform:               platform,
				ToolchainExplicit:      cmd.Flags().Changed("toolchain"),
				EnvironmentExplicit:    cmd.Flags().Changed("environment"),
				PlatformExplicit:       cmd.Flags().Changed("platform"),
				PublishSpecs:           publishSpecs,
				Recreate:               recreate,
				PullOutput:             cmd.ErrOrStderr(),
				Progress: func(message string) {
					_ = stderrUI.Progress(message)
				},
			}
			started := time.Now()
			result, err := local.Up(ctx, request)
			var detectionErr *workspace.ToolchainDetectionError
			if err != nil && errors.As(err, &detectionErr) && !outputJSON && isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()) {
				selected, selectErr := promptToolchainSelection(cmd.InOrStdin(), cmd.OutOrStdout(), detectionErr.Matches)
				if selectErr != nil {
					return selectErr
				}
				request.Toolchain = string(selected)
				request.ToolchainExplicit = true
				result, err = local.Up(ctx, request)
			}
			if err != nil {
				return err
			}
			if err := writeWorkspaceRecord(result); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if outputJSON {
				return writeJSON(out, upPayload(result, time.Since(started)))
			}
			if err := stdoutUI.Success(fmt.Sprintf("Workspace %s in %s", displayUpAction(result.Action), formatDuration(time.Since(started)))); err != nil {
				return err
			}
			if err := stdoutUI.Fields(
				cliui.Field{Label: "workspace", Value: result.Project.Slug},
				cliui.Field{Label: "environment", Value: displayUpEnvironment(result.Environment)},
				cliui.Field{Label: "toolchain", Value: displayToolchain(string(result.Environment.Toolchain))},
				cliui.Field{Label: "platform", Value: result.Environment.Platform},
				cliui.Field{Label: "project", Value: result.Project.ProjectDir},
			); err != nil {
				return err
			}
			if err := stdoutUI.Next("elyro shell", "elyro exec -- <command>", "elyro open"); err != nil {
				return err
			}
			if openEditor {
				return openCurrentWorkspace(cmd, opts, editorName, false)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&toolchain, "toolchain", "", "Toolchain to start (python, go, java, or node); detected when omitted")
	cmd.Flags().StringVar(&environmentName, "environment", "", "Project-defined environment from elyro.yaml to start")
	cmd.Flags().StringVar(&platform, "platform", workspace.DefaultPlatform, "Target container platform (linux/amd64 or linux/arm64)")
	cmd.Flags().BoolVar(&allowUnsafeEnvironment, "allow-unsafe-environment", false, "Allow privileged mode, Docker socket, or host mounts outside the project")
	cmd.Flags().StringArrayVar(&publishSpecs, "publish", nil, "Publish a local port to the workspace container. Repeatable; supports <port> or <host-port>:<container-port>")
	cmd.Flags().BoolVar(&openEditor, "open", false, "Open the workspace in an editor after it is ready")
	cmd.Flags().StringVar(&editorName, "editor", "", "Editor to open with --open: cursor or code")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print the workspace result as JSON")
	cmd.Flags().BoolVar(&recreate, "recreate", false, "Recreate an existing workspace before starting")
	return cmd
}

func displayUpAction(action local.WorkspaceAction) string {
	switch action {
	case local.WorkspaceActionCreated, local.WorkspaceActionRecreated, local.WorkspaceActionStarted, local.WorkspaceActionReused:
		return string(action)
	default:
		return "ready"
	}
}

type upJSONView struct {
	SchemaVersion int                   `json:"schema_version"`
	Kind          string                `json:"kind"`
	Action        local.WorkspaceAction `json:"action"`
	DurationMS    int64                 `json:"duration_ms"`
	Workspace     workspaceJSONView     `json:"workspace"`
}

func upPayload(result local.UpResult, duration time.Duration) upJSONView {
	return upJSONView{
		SchemaVersion: 1,
		Kind:          "workspace",
		Action:        result.Action,
		DurationMS:    duration.Milliseconds(),
		Workspace:     workspacePayload(result.Project, &result.Container),
	}
}

func displayUpEnvironment(environment workspace.ResolvedEnvironment) string {
	if name := strings.TrimSpace(environment.Name); name != "" {
		return name
	}
	if toolchain := strings.TrimSpace(string(environment.Toolchain)); toolchain != "" {
		return toolchain
	}
	return "custom"
}

func formatDuration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", duration.Seconds())
}

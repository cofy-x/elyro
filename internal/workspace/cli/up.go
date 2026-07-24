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
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start or reuse a Workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectRoot, err := resolvedProjectRoot(cmd, opts)
			if err != nil {
				return err
			}
			stdoutUI := cliui.New(cmd.OutOrStdout())
			stderrUI := cliui.New(cmd.ErrOrStderr())
			if dryRun && (openEditor || cmd.Flags().Changed("editor")) {
				return errors.New("--dry-run cannot be combined with --open or --editor")
			}
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
				ProjectDir:             projectRoot.Dir,
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
			if dryRun {
				plan, planErr := local.PlanUp(ctx, request)
				var detectionErr *workspace.ToolchainDetectionError
				if planErr != nil && errors.As(planErr, &detectionErr) && !outputJSON && isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()) {
					selected, selectErr := promptToolchainSelection(cmd.InOrStdin(), cmd.OutOrStdout(), detectionErr.Matches)
					if selectErr != nil {
						return selectErr
					}
					request.Toolchain = string(selected)
					request.ToolchainExplicit = true
					plan, planErr = local.PlanUp(ctx, request)
				}
				if planErr != nil {
					return planErr
				}
				if outputJSON {
					return writeJSON(cmd.OutOrStdout(), upPlanPayload(plan, projectRoot))
				}
				return printUpPlan(stdoutUI, plan, projectRoot, request, cmd.Flags().Changed("project-dir"))
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
			fields := []cliui.Field{}
			if result.Action == local.WorkspaceActionRecreated {
				fields = append(fields, cliui.Field{Label: "reason", Value: displayPlanReasons(result.Reasons)})
			}
			fields = append(fields,
				cliui.Field{Label: "workspace", Value: result.Project.Slug},
				cliui.Field{Label: "environment", Value: displayUpEnvironment(result.Environment)},
				cliui.Field{Label: "toolchain", Value: displayToolchain(string(result.Environment.Toolchain))},
				cliui.Field{Label: "platform", Value: result.Environment.Platform},
				cliui.Field{Label: "project", Value: result.Project.ProjectDir},
			)
			if err := stdoutUI.Fields(fields...); err != nil {
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

	cmd.Flags().StringVar(&toolchain, "toolchain", "", "Toolchain to start (python, go, or node); detected when omitted")
	cmd.Flags().StringVar(&environmentName, "environment", "", "Project-defined environment from elyro.yaml to start")
	cmd.Flags().StringVar(&platform, "platform", workspace.DefaultPlatform, "Target container platform (linux/amd64 or linux/arm64)")
	cmd.Flags().BoolVar(&allowUnsafeEnvironment, "allow-unsafe-environment", false, "Allow privileged mode, Docker socket, or host mounts outside the project")
	cmd.Flags().StringArrayVar(&publishSpecs, "publish", nil, "Publish a local port to the workspace container. Repeatable; supports <port> or <host-port>:<container-port>")
	cmd.Flags().BoolVar(&openEditor, "open", false, "Open the workspace in an editor after it is ready")
	cmd.Flags().StringVar(&editorName, "editor", "", "Editor to open with --open: cursor or code")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print the workspace result as JSON")
	cmd.Flags().BoolVar(&recreate, "recreate", false, "Recreate an existing workspace before starting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the workspace action without changing local state")
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
	SchemaVersion int                           `json:"schema_version"`
	Kind          string                        `json:"kind"`
	Action        local.WorkspaceAction         `json:"action"`
	Reasons       []local.WorkspaceChangeReason `json:"reasons"`
	DurationMS    int64                         `json:"duration_ms"`
	Workspace     workspaceJSONView             `json:"workspace"`
}

func upPayload(result local.UpResult, duration time.Duration) upJSONView {
	return upJSONView{
		SchemaVersion: 1,
		Kind:          "workspace",
		Action:        result.Action,
		Reasons:       result.Reasons,
		DurationMS:    duration.Milliseconds(),
		Workspace:     workspacePayload(result.Project, &result.Container),
	}
}

type workspacePlanProjectView struct {
	Root   string                      `json:"root"`
	Source workspace.ProjectRootSource `json:"source"`
}

type workspacePlanImageView struct {
	Reference string                     `json:"reference"`
	Status    local.WorkspaceImageStatus `json:"status"`
}

type workspacePlanWorkspaceView struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ProjectDir     string   `json:"project_dir"`
	MountDir       string   `json:"mount_dir"`
	CurrentStatus  string   `json:"current_status"`
	Environment    string   `json:"environment"`
	Toolchain      string   `json:"toolchain,omitempty"`
	Platform       string   `json:"platform"`
	Hostname       string   `json:"hostname"`
	PublishedPorts []string `json:"published_ports"`
}

type upPlanJSONView struct {
	SchemaVersion int                           `json:"schema_version"`
	Kind          string                        `json:"kind"`
	Operation     string                        `json:"operation"`
	Action        local.WorkspacePlanAction     `json:"action"`
	Reasons       []local.WorkspaceChangeReason `json:"reasons"`
	Project       workspacePlanProjectView      `json:"project"`
	Image         workspacePlanImageView        `json:"image"`
	Workspace     workspacePlanWorkspaceView    `json:"workspace"`
}

func upPlanPayload(plan local.UpPlan, root workspace.ProjectRoot) upPlanJSONView {
	status := "absent"
	if plan.Container != nil {
		status = plan.Container.Status
	}
	published := []string{}
	if plan.Published != "" {
		published = strings.Split(plan.Published, ",")
	}
	return upPlanJSONView{
		SchemaVersion: 1,
		Kind:          "workspace_plan",
		Operation:     "up",
		Action:        plan.Action,
		Reasons:       plan.Reasons,
		Project:       workspacePlanProjectView{Root: root.Dir, Source: root.Source},
		Image:         workspacePlanImageView{Reference: plan.Environment.Image, Status: plan.ImageStatus},
		Workspace: workspacePlanWorkspaceView{
			ID: plan.Project.ProjectHash, Name: plan.Project.Slug, ProjectDir: plan.Project.ProjectDir,
			MountDir: plan.Project.MountDir, CurrentStatus: status, Environment: displayUpEnvironment(plan.Environment),
			Toolchain: string(plan.Environment.Toolchain), Platform: plan.Environment.Platform, Hostname: plan.Project.Slug,
			PublishedPorts: published,
		},
	}
}

func printUpPlan(ui cliui.Renderer, plan local.UpPlan, root workspace.ProjectRoot, request local.UpRequest, projectDirExplicit bool) error {
	if err := ui.Success("Workspace plan ready"); err != nil {
		return err
	}
	fields := []cliui.Field{
		{Label: "action", Value: string(plan.Action)},
		{Label: "reason", Value: displayPlanReasons(plan.Reasons)},
		{Label: "workspace", Value: plan.Project.Slug},
		{Label: "environment", Value: displayUpEnvironment(plan.Environment)},
		{Label: "toolchain", Value: displayToolchain(string(plan.Environment.Toolchain))},
		{Label: "platform", Value: plan.Environment.Platform},
		{Label: "image", Value: plan.Environment.Image + " (" + string(plan.ImageStatus) + ")"},
		{Label: "project", Value: root.Dir + " (" + string(root.Source) + ")"},
	}
	if err := ui.Fields(fields...); err != nil {
		return err
	}
	return ui.Next(upCommandForPlan(request, projectDirExplicit))
}

func upCommandForPlan(request local.UpRequest, projectDirExplicit bool) string {
	args := []string{"elyro", "up"}
	if projectDirExplicit {
		args = append(args, "--project-dir", quoteCommandArg(request.ProjectDir))
	}
	if request.EnvironmentExplicit {
		args = append(args, "--environment", quoteCommandArg(request.Environment))
	}
	if request.ToolchainExplicit {
		args = append(args, "--toolchain", quoteCommandArg(request.Toolchain))
	}
	if request.PlatformExplicit {
		args = append(args, "--platform", quoteCommandArg(request.Platform))
	}
	for _, publish := range request.PublishSpecs {
		args = append(args, "--publish", quoteCommandArg(publish))
	}
	if request.AllowUnsafeEnvironment {
		args = append(args, "--allow-unsafe-environment")
	}
	if request.Recreate {
		args = append(args, "--recreate")
	}
	return strings.Join(args, " ")
}

func quoteCommandArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("/_:.,@+=-", r)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func displayPlanReasons(reasons []local.WorkspaceChangeReason) string {
	if len(reasons) == 0 {
		return "none"
	}
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		values = append(values, strings.ReplaceAll(string(reason), "_", " "))
	}
	return strings.Join(values, ", ")
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

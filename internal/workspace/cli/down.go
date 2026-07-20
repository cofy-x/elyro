package cli

import (
	"context"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

func newDownCmd(opts *GlobalOptions) *cobra.Command {
	var outputJSON bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Remove the current Workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectRoot, err := resolvedProjectRoot(cmd, opts)
			if err != nil {
				return err
			}
			ui := cliui.New(cmd.OutOrStdout())
			sshConfigPath, err := expandSSHConfigPath(opts.SSHConfigPath)
			if err != nil {
				return err
			}

			ctx := context.Background()
			request := local.DownRequest{
				ProjectDir:    projectRoot.Dir,
				SSHConfigPath: sshConfigPath,
			}
			plan, err := resolvedDownPlan(ctx, request)
			if err != nil {
				return err
			}
			if dryRun {
				if outputJSON {
					return writeJSON(cmd.OutOrStdout(), downPlanPayload(plan, projectRoot))
				}
				return printDownPlan(ui, plan, projectRoot, cmd.Flags().Changed("project-dir"))
			}
			result, err := local.ApplyDown(ctx, request, plan)
			if err != nil {
				return err
			}
			if err := removeWorkspaceRecord(result.Project.ProjectDir); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if outputJSON {
				return writeJSON(out, downJSONView{
					SchemaVersion: 1,
					Kind:          "workspace",
					Workspace:     removedWorkspacePayload(result),
					Removed:       result.Container != nil,
				})
			}
			if result.Container != nil {
				if err := ui.Success("Workspace removed"); err != nil {
					return err
				}
			} else {
				if err := ui.Warning("Workspace was not running"); err != nil {
					return err
				}
			}
			return ui.Fields(
				cliui.Field{Label: "workspace", Value: result.Project.Slug},
				cliui.Field{Label: "project", Value: result.Project.ProjectDir},
				cliui.Field{Label: "preserved", Value: "project files, mounted host data, local images"},
			)
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print the removal result as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workspace removal without changing local state")
	return cmd
}

func resolvedDownPlan(ctx context.Context, request local.DownRequest) (local.DownPlan, error) {
	preliminary, err := local.PlanDown(ctx, local.DownPlanRequest{DownRequest: request})
	if err != nil {
		return local.DownPlan{}, err
	}
	alias := preliminary.Project.HostAlias
	if preliminary.Container != nil && preliminary.Container.HostAlias != "" {
		alias = preliminary.Container.HostAlias
	}
	managedSSH, err := elyroworkspace.HasManagedSSHHost(request.SSHConfigPath, alias)
	if err != nil {
		return local.DownPlan{}, err
	}
	knownHost, err := elyroworkspace.HasKnownSSHHost(elyroworkspace.DefaultKnownHostsFile(), alias)
	if err != nil {
		return local.DownPlan{}, err
	}
	store, _, err := loadWorkspaceStore()
	if err != nil {
		return local.DownPlan{}, err
	}
	registry, err := elyroworkspace.HasWorkspaceRecord(store, preliminary.Project.ProjectDir)
	if err != nil {
		return local.DownPlan{}, err
	}
	return local.CompleteDownPlan(preliminary, managedSSH, knownHost, registry), nil
}

func removeWorkspaceRecord(projectDir string) error {
	path, err := elyroworkspace.DefaultPath()
	if err != nil {
		return err
	}
	return elyroworkspace.RemoveFile(path, projectDir)
}

type downJSONView struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Workspace     workspaceJSONView `json:"workspace"`
	Removed       bool              `json:"removed"`
}

type downPlanJSONView struct {
	SchemaVersion int                                `json:"schema_version"`
	Kind          string                             `json:"kind"`
	Operation     string                             `json:"operation"`
	Action        local.WorkspacePlanAction          `json:"action"`
	Project       workspacePlanProjectView           `json:"project"`
	Workspace     workspaceJSONView                  `json:"workspace"`
	Removes       []local.WorkspaceManagedResource   `json:"removes"`
	Preserves     []local.WorkspacePreservedResource `json:"preserves"`
}

func downPlanPayload(plan local.DownPlan, root elyroworkspace.ProjectRoot) downPlanJSONView {
	return downPlanJSONView{
		SchemaVersion: 1, Kind: "workspace_plan", Operation: "down", Action: plan.Action,
		Project:   workspacePlanProjectView{Root: root.Dir, Source: root.Source},
		Workspace: workspacePayload(plan.Project, plan.Container), Removes: plan.Removes, Preserves: plan.Preserves,
	}
}

func printDownPlan(ui cliui.Renderer, plan local.DownPlan, root elyroworkspace.ProjectRoot, projectDirExplicit bool) error {
	if err := ui.Warning("Workspace removal plan ready"); err != nil {
		return err
	}
	removes := "none"
	if len(plan.Removes) > 0 {
		values := make([]string, 0, len(plan.Removes))
		for _, resource := range plan.Removes {
			values = append(values, strings.ReplaceAll(string(resource), "_", " "))
		}
		removes = strings.Join(values, ", ")
	}
	if err := ui.Fields(
		cliui.Field{Label: "action", Value: string(plan.Action)},
		cliui.Field{Label: "workspace", Value: plan.Project.Slug},
		cliui.Field{Label: "removes", Value: removes},
		cliui.Field{Label: "preserves", Value: "project files, mounted host data, local images"},
		cliui.Field{Label: "project", Value: root.Dir + " (" + string(root.Source) + ")"},
	); err != nil {
		return err
	}
	if plan.Action == local.WorkspacePlanActionNone {
		return nil
	}
	next := "elyro down"
	if projectDirExplicit {
		next += " --project-dir " + quoteCommandArg(root.Dir)
	}
	return ui.Next(next)
}

func removedWorkspacePayload(result local.DownResult) workspaceJSONView {
	view := workspacePayload(result.Project, result.Container)
	if result.Container != nil {
		view.Status = "removed"
	}
	return view
}

package cli

import (
	"context"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

func newStatusCmd(opts *GlobalOptions) *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the Elyro workspace status for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ui := cliui.New(cmd.OutOrStdout())
			ctx := context.Background()
			result, err := local.Status(ctx, local.StatusRequest{
				ProjectDir: opts.ProjectDir,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if outputJSON {
				return writeJSON(out, localStatusPayload(result))
			}
			projectCtx := result.Project
			info := result.Container
			if info == nil {
				if err := ui.Warning("Workspace is not running"); err != nil {
					return err
				}
				if err := ui.Fields(
					cliui.Field{Label: "project", Value: projectCtx.ProjectDir},
					cliui.Field{Label: "workspace", Value: projectCtx.Slug},
				); err != nil {
					return err
				}
				return ui.Next("elyro up")
			}

			if err := ui.Success("Workspace is running"); err != nil {
				return err
			}
			return ui.Fields(
				cliui.Field{Label: "workspace", Value: projectCtx.Slug},
				cliui.Field{Label: "project", Value: info.ProjectDir},
				cliui.Field{Label: "hostname", Value: info.Hostname},
				cliui.Field{Label: "environment", Value: displayEnvironment(info.Environment, info.Toolchain)},
				cliui.Field{Label: "toolchain", Value: displayToolchain(info.Toolchain)},
				cliui.Field{Label: "platform", Value: displayPlatform(info.Platform)},
				cliui.Field{Label: "image", Value: info.Image},
				cliui.Field{Label: "privileged", Value: displayPrivileged(info.Privileged)},
				cliui.Field{Label: "published", Value: displayOptional(info.Published)},
				cliui.Field{Label: "mounts", Value: displayOptional(info.Mounts)},
				cliui.Field{Label: "mount", Value: projectCtx.MountDir},
			)
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print status as JSON")
	return cmd
}

type localStatusPayloadView struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Workspace     workspaceJSONView `json:"workspace"`
}

func localStatusPayload(result local.StatusResult) localStatusPayloadView {
	payload := localStatusPayloadView{
		SchemaVersion: 1,
		Kind:          "workspace",
		Workspace:     workspacePayload(result.Project, result.Container),
	}
	return payload
}

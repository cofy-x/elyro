package cli

import (
	"context"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

func newDownCmd(opts *GlobalOptions) *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Remove the current Workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := resolvedProjectDir(cmd, opts)
			if err != nil {
				return err
			}
			ui := cliui.New(cmd.OutOrStdout())
			sshConfigPath, err := expandSSHConfigPath(opts.SSHConfigPath)
			if err != nil {
				return err
			}

			ctx := context.Background()
			result, err := local.Down(ctx, local.DownRequest{
				ProjectDir:    projectDir,
				SSHConfigPath: sshConfigPath,
			})
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
			)
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print the removal result as JSON")
	return cmd
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

func removedWorkspacePayload(result local.DownResult) workspaceJSONView {
	view := workspacePayload(result.Project, result.Container)
	if result.Container != nil {
		view.Status = "removed"
	}
	return view
}

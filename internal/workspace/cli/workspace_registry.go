package cli

import (
	"context"
	"io"
	"time"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

func newListCmd(opts *GlobalOptions) *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List known local Elyro workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, _, err := loadWorkspaceStore()
			if err != nil {
				return err
			}
			current, currentOK := currentWorkspaceFromStore(store, opts.ProjectDir)
			items, err := liveWorkspaceItems(context.Background(), store.Workspaces, current, currentOK)
			if err != nil {
				return err
			}
			if outputJSON {
				return writeJSON(cmd.OutOrStdout(), workspaceListPayload(items))
			}
			return printWorkspaceList(cmd.OutOrStdout(), items)
		},
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print workspace list as JSON")
	return cmd
}

func writeWorkspaceRecord(result local.UpResult) error {
	path, err := elyroworkspace.DefaultPath()
	if err != nil {
		return err
	}
	return elyroworkspace.UpsertFile(path, workspaceRecord(result))
}

func workspaceRecord(result local.UpResult) elyroworkspace.Record {
	return elyroworkspace.Record{
		Name:                  result.Project.Slug,
		Kind:                  elyroworkspace.KindWorkspace,
		ProjectDir:            result.Project.ProjectDir,
		HostWorkspaceDir:      result.Project.ProjectDir,
		ContainerWorkspaceDir: result.Project.MountDir,
		ContainerName:         result.Container.Name,
		Hostname:              result.Container.Hostname,
		SSHAlias:              result.Container.HostAlias,
		Environment:           result.Environment.Name,
		Toolchain:             string(result.Environment.Toolchain),
		Platform:              result.Environment.Platform,
		UpdatedAt:             time.Now().UTC(),
	}
}

func loadWorkspaceStore() (elyroworkspace.Store, string, error) {
	path, err := elyroworkspace.DefaultPath()
	if err != nil {
		return elyroworkspace.Store{}, "", err
	}
	store, err := elyroworkspace.Load(path)
	if err != nil {
		return elyroworkspace.Store{}, "", err
	}
	return store, path, nil
}

func currentWorkspaceFromStore(store elyroworkspace.Store, projectDir string) (elyroworkspace.Record, bool) {
	record, err := elyroworkspace.Current(store, projectDir)
	return record, err == nil
}

func printWorkspaceList(out io.Writer, items []workspaceListItem) error {
	ui := cliui.New(out)
	if len(items) == 0 {
		if err := ui.Warning("No workspaces found"); err != nil {
			return err
		}
		return ui.Next("elyro up")
	}
	if err := ui.Title("Workspaces"); err != nil {
		return err
	}
	printed := false
	for _, item := range items {
		if printed {
			if err := ui.Text(""); err != nil {
				return err
			}
		}
		if err := printWorkspace(ui, item.Workspace, item.Current); err != nil {
			return err
		}
		printed = true
	}
	return nil
}

func printWorkspace(ui cliui.Renderer, view workspaceJSONView, current bool) error {
	marker := "workspace"
	if current {
		marker = "workspace *"
	}
	return ui.Fields(
		cliui.Field{Label: marker, Value: view.Name},
		cliui.Field{Label: "status", Value: view.Status},
		cliui.Field{Label: "project", Value: view.ProjectDir},
		cliui.Field{Label: "hostname", Value: displayOptional(view.Hostname)},
		cliui.Field{Label: "environment", Value: displayOptional(view.Environment)},
		cliui.Field{Label: "toolchain", Value: displayOptional(view.Toolchain)},
		cliui.Field{Label: "platform", Value: displayOptional(view.Platform)},
		cliui.Field{Label: "mount", Value: view.MountDir},
	)
}

type workspaceListView struct {
	SchemaVersion int                 `json:"schema_version"`
	Kind          string              `json:"kind"`
	Workspaces    []workspaceListItem `json:"workspaces"`
}

type workspaceListItem struct {
	Workspace workspaceJSONView `json:"workspace"`
	Current   bool              `json:"current"`
}

func liveWorkspaceItems(ctx context.Context, records []elyroworkspace.Record, current elyroworkspace.Record, currentOK bool) ([]workspaceListItem, error) {
	items := make([]workspaceListItem, 0, len(records))
	for _, record := range records {
		if record.Kind != elyroworkspace.KindWorkspace {
			continue
		}
		isCurrent := currentOK && record.Kind == current.Kind && record.Name == current.Name && record.ProjectDir == current.ProjectDir
		result, err := local.Status(ctx, local.StatusRequest{ProjectDir: record.ProjectDir})
		if err != nil {
			return nil, err
		}
		items = append(items, workspaceListItem{Workspace: workspacePayload(result.Project, result.Container), Current: isCurrent})
	}
	return items, nil
}

func workspaceListPayload(items []workspaceListItem) workspaceListView {
	return workspaceListView{SchemaVersion: 1, Kind: "workspace_list", Workspaces: items}
}

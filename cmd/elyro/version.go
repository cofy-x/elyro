package main

import (
	"encoding/json"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroversion "github.com/cofy-x/elyro/internal/version"
	"github.com/spf13/cobra"
)

type versionView struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func newVersionCmd() *cobra.Command {
	var outputJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print Elyro build version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			view := versionView{
				Version:   elyroversion.ReleaseVersion(),
				Commit:    elyroversion.Commit,
				BuildDate: elyroversion.BuildDate,
			}
			if outputJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(view)
			}
			ui := cliui.New(cmd.OutOrStdout())
			if err := ui.Title("Elyro " + view.Version); err != nil {
				return err
			}
			return ui.Fields(
				cliui.Field{Label: "commit", Value: view.Commit},
				cliui.Field{Label: "built", Value: view.BuildDate},
			)
		},
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print machine-readable JSON")
	return cmd
}

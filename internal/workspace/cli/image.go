package cli

import "github.com/spf13/cobra"

// NewImageCmd creates the project-owned Workspace image command group.
func NewImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage the project Workspace image",
	}
	cmd.AddCommand(newImageInitCmd(), newImageBuildCmd())
	return cmd
}

package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/editor"
	"github.com/spf13/cobra"
)

func newOpenCmd(opts *GlobalOptions) *cobra.Command {
	var editorName string
	var printOnly bool

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open the current Elyro workspace in an editor",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return openCurrentWorkspace(cmd, opts, editorName, printOnly)
		},
	}
	cmd.Flags().StringVar(&editorName, "editor", "", "Editor to open: cursor, code, or vscode")
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print editor open commands without launching")
	return cmd
}

func openCurrentWorkspace(cmd *cobra.Command, opts *GlobalOptions, editorName string, printOnly bool) error {
	projectDir, err := resolvedProjectDir(cmd, opts)
	if err != nil {
		return err
	}
	store, _, err := loadWorkspaceStore()
	if err != nil {
		return err
	}
	record, err := elyroworkspace.Current(store, projectDir)
	if err != nil {
		if errors.Is(err, elyroworkspace.ErrNoCurrent) {
			return errors.New("no current workspace found; run `elyro up` from this project first")
		}
		return err
	}
	if printOnly {
		return printEditorOpenHelpHost(cmd.OutOrStdout(), record.SSHAlias, record.ContainerWorkspaceDir)
	}
	option, err := resolveEditorOption(cmd.InOrStdin(), cmd.OutOrStdout(), editorName, record.SSHAlias, record.ContainerWorkspaceDir)
	if err != nil {
		_ = printEditorOpenHelpHost(cmd.OutOrStdout(), record.SSHAlias, record.ContainerWorkspaceDir)
		return err
	}
	if err := editor.Open(option); err != nil {
		return fmt.Errorf("open %s: %w", option.Label, err)
	}
	return cliui.New(cmd.OutOrStdout()).Success(fmt.Sprintf("Opened %s in %s", record.Name, option.Label))
}

func resolveEditorOption(in io.Reader, out io.Writer, name, hostAlias, remoteDir string) (editor.Option, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized != "" {
		switch normalized {
		case "cursor":
			return editor.Option{
				Label:      "Cursor",
				BinaryName: "cursor",
				FolderURI:  editor.RemoteSSHFolderURI(hostAlias, remoteDir),
				Command:    editor.NewWindowCommand("cursor", hostAlias, remoteDir),
			}, nil
		case "code", "vscode", "vs-code":
			return editor.Option{
				Label:      "VS Code",
				BinaryName: "code",
				FolderURI:  editor.RemoteSSHFolderURI(hostAlias, remoteDir),
				Command:    editor.NewWindowCommand("code", hostAlias, remoteDir),
			}, nil
		default:
			return editor.Option{}, fmt.Errorf("unsupported editor %q (supported: cursor, code, vscode)", name)
		}
	}

	options := editor.DetectOptions(hostAlias, remoteDir)
	if len(options) == 0 {
		return editor.Option{}, errors.New("no supported editor binary found in PATH; install Cursor or VS Code, or use --print")
	}
	if isInteractive(in, out) && len(options) > 1 {
		selected := promptEditorSelection(in, out, options)
		if selected >= 0 {
			return options[selected], nil
		}
	}
	return options[0], nil
}

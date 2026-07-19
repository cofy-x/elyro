package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace/editor"
)

func isInteractive(in io.Reader, out io.Writer) bool {
	stdin, stdinOK := in.(*os.File)
	stdout, stdoutOK := out.(*os.File)
	if !stdinOK || !stdoutOK {
		return false
	}
	stdinInfo, stdinErr := stdin.Stat()
	stdoutInfo, stdoutErr := stdout.Stat()
	return stdinErr == nil && stdoutErr == nil && stdinInfo.Mode()&os.ModeCharDevice != 0 && stdoutInfo.Mode()&os.ModeCharDevice != 0
}

var errEditorSelectionCancelled = errors.New("editor selection cancelled")

func promptEditorSelection(in io.Reader, out io.Writer, options []editor.Option) (int, error) {
	if len(options) == 0 {
		return -1, errors.New("no editor options available")
	}

	reader := bufio.NewReader(in)
	for attempt := 0; attempt < 2; attempt++ {
		fmt.Fprintln(out)
		ui := cliui.New(out)
		if err := ui.Question("Choose an editor [" + options[0].Label + "]"); err != nil {
			return -1, err
		}
		for i, option := range options {
			fmt.Fprintf(out, "  %d  %s\n", i+1, option.Label)
		}
		fmt.Fprintln(out, "  q  Cancel")
		if err := ui.Prompt("Select: "); err != nil {
			return -1, err
		}

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return -1, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return 0, nil
		}
		if strings.EqualFold(line, "q") || strings.EqualFold(line, "cancel") {
			return -1, errEditorSelectionCancelled
		}

		selected, convErr := strconv.Atoi(line)
		if convErr == nil && selected >= 1 && selected <= len(options) {
			return selected - 1, nil
		}

		fmt.Fprintln(out, "Invalid selection; choose an editor number or q to cancel.")
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return -1, errors.New("invalid editor selection")
}

func printEditorOpenHelpHost(out io.Writer, hostAlias, remoteDir string) error {
	ui := cliui.New(out)
	if err := ui.Section("Open in editor"); err != nil {
		return err
	}
	if err := ui.Fields(
		cliui.Field{Label: "VS Code", Value: editor.RemoteSSHOpenURI("vscode", hostAlias, remoteDir)},
		cliui.Field{Label: "Cursor", Value: editor.RemoteSSHOpenURI("cursor", hostAlias, remoteDir)},
	); err != nil {
		return err
	}
	return ui.Next(
		editor.NewWindowCommand("code", hostAlias, remoteDir),
		editor.NewWindowCommand("cursor", hostAlias, remoteDir),
	)
}

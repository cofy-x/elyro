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

func promptEditorSelection(in io.Reader, out io.Writer, options []editor.Option) int {
	if len(options) == 0 {
		return -1
	}

	reader := bufio.NewReader(in)
	maxChoice := len(options) + 1
	for attempt := 0; attempt < 2; attempt++ {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Open editor now?")
		for i, option := range options {
			fmt.Fprintf(out, "  %d. %s\n", i+1, option.Label)
		}
		fmt.Fprintf(out, "  %d. Skip\n", maxChoice)
		fmt.Fprint(out, "Select an option and press Enter (default: Skip): ")

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return -1
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return -1
		}

		selected, convErr := strconv.Atoi(line)
		if convErr == nil {
			switch {
			case selected >= 1 && selected <= len(options):
				return selected - 1
			case selected == maxChoice:
				return -1
			}
		}

		fmt.Fprintln(out, "Invalid selection.")
		if errors.Is(err, io.EOF) {
			return -1
		}
	}
	return -1
}

func printEditorOpenHelpHost(out io.Writer, hostAlias, remoteDir string) error {
	ui := cliui.New(out)
	if err := ui.Title("Open in editor"); err != nil {
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

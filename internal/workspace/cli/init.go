package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace"
)

type InitProjectOptions struct {
	ProjectDir  string
	Toolchain   string
	Yes         bool
	In          io.Reader
	Out         io.Writer
	Interactive bool
}

func InitProject(options InitProjectOptions) error {
	if options.In == nil {
		options.In = strings.NewReader("")
	}
	if options.Out == nil {
		options.Out = io.Discard
	}
	reader := bufferedReader(options.In)
	ui := cliui.New(options.Out)

	if _, err := workspace.ProjectConfigPath(options.ProjectDir); err != nil {
		return err
	}
	if _, err := workspace.ValidateProjectConfigTarget(options.ProjectDir); err != nil {
		return err
	}
	toolchain, err := initToolchain(options.ProjectDir, options.Toolchain, options.Interactive, reader, options.Out)
	if err != nil {
		return err
	}

	if !options.Yes {
		if !options.Interactive {
			return errors.New("refusing to write elyro.yaml in a non-interactive session; pass --yes")
		}
		confirmed, err := promptConfirmation(reader, options.Out, fmt.Sprintf("Create elyro.yaml with Toolchain %s?", displayToolchainChoice(toolchain)))
		if err != nil {
			return err
		}
		if !confirmed {
			return ui.Warning("Project initialization cancelled")
		}
	}

	writtenPath, err := workspace.WriteProjectConfig(options.ProjectDir, toolchain)
	if err != nil {
		return err
	}
	if err := ui.Success("Created elyro.yaml"); err != nil {
		return err
	}
	if err := ui.Fields(
		cliui.Field{Label: "config", Value: writtenPath},
		cliui.Field{Label: "toolchain", Value: string(toolchain)},
	); err != nil {
		return err
	}
	return ui.Next("elyro up")
}

func initToolchain(projectDir, explicit string, interactive bool, in io.Reader, out io.Writer) (workspace.Toolchain, error) {
	if strings.TrimSpace(explicit) != "" {
		return workspace.ParseToolchain(strings.TrimSpace(explicit))
	}
	detected, err := workspace.DetectToolchain(projectDir)
	if err == nil {
		return detected, nil
	}
	var detectionErr *workspace.ToolchainDetectionError
	if !errors.As(err, &detectionErr) || !interactive {
		return "", err
	}
	return promptToolchainSelection(in, out, detectionErr.Matches)
}

func promptToolchainSelection(in io.Reader, out io.Writer, detected []workspace.Toolchain) (workspace.Toolchain, error) {
	ui := cliui.New(out)
	if len(detected) == 0 {
		if err := ui.Warning("No project language was detected"); err != nil {
			return "", err
		}
	} else {
		if err := ui.Warning("Multiple project languages were detected"); err != nil {
			return "", err
		}
	}
	if err := ui.Question("Choose a Toolchain"); err != nil {
		return "", err
	}
	choices := []workspace.Toolchain{workspace.ToolchainPython, workspace.ToolchainGo, workspace.ToolchainJava, workspace.ToolchainNode}
	for i, toolchain := range choices {
		fmt.Fprintf(out, "  %d  %s\n", i+1, displayToolchainChoice(toolchain))
	}
	if err := ui.Prompt("Select: "); err != nil {
		return "", err
	}
	line, err := bufferedReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	switch strings.TrimSpace(line) {
	case "1", "python":
		return workspace.ToolchainPython, nil
	case "2", "go":
		return workspace.ToolchainGo, nil
	case "3", "java":
		return workspace.ToolchainJava, nil
	case "4", "node":
		return workspace.ToolchainNode, nil
	default:
		return "", errors.New("invalid toolchain selection; pass --toolchain python, go, java, or node")
	}
}

func promptConfirmation(in io.Reader, out io.Writer, message string) (bool, error) {
	if err := cliui.New(out).Prompt(message + " [y/N] "); err != nil {
		return false, err
	}
	line, err := bufferedReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func displayToolchainChoice(toolchain workspace.Toolchain) string {
	switch toolchain {
	case workspace.ToolchainPython:
		return "Python"
	case workspace.ToolchainGo:
		return "Go"
	case workspace.ToolchainJava:
		return "Java"
	case workspace.ToolchainNode:
		return "Node.js"
	default:
		return string(toolchain)
	}
}

func bufferedReader(in io.Reader) *bufio.Reader {
	if reader, ok := in.(*bufio.Reader); ok {
		return reader
	}
	return bufio.NewReader(in)
}

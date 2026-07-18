package workspace

import (
	"fmt"

	elyroimages "github.com/cofy-x/elyro/internal/images"
)

type Toolchain string

const (
	ToolchainPython Toolchain = "python"
	ToolchainGo     Toolchain = "go"
	ToolchainJava   Toolchain = "java"
	ToolchainNode   Toolchain = "node"
)

const (
	BaseImageRepo       = "elyro/workspace-base"
	WorkspacePythonRepo = "elyro/workspace-python"
	WorkspaceGoRepo     = "elyro/workspace-go"
	WorkspaceJavaRepo   = "elyro/workspace-java"
	WorkspaceNodeRepo   = "elyro/workspace-node"
)

func ParseToolchain(raw string) (Toolchain, error) {
	switch Toolchain(raw) {
	case ToolchainPython:
		return ToolchainPython, nil
	case ToolchainGo:
		return ToolchainGo, nil
	case ToolchainJava:
		return ToolchainJava, nil
	case ToolchainNode:
		return ToolchainNode, nil
	default:
		return "", fmt.Errorf("unsupported toolchain %q (supported: python, go, java, node)", raw)
	}
}

func (f Toolchain) Image(platform string) string {
	switch f {
	case ToolchainPython:
		return elyroimages.Reference(WorkspacePythonRepo, platform)
	case ToolchainGo:
		return elyroimages.Reference(WorkspaceGoRepo, platform)
	case ToolchainJava:
		return elyroimages.Reference(WorkspaceJavaRepo, platform)
	case ToolchainNode:
		return elyroimages.Reference(WorkspaceNodeRepo, platform)
	default:
		return ""
	}
}

func BaseImage(platform string) string {
	return elyroimages.Reference(BaseImageRepo, platform)
}

// DockerContextDir is the repository-relative directory passed to docker build as context.
func (f Toolchain) DockerContextDir() string {
	switch f {
	case ToolchainPython:
		return "images/workspace-python"
	case ToolchainGo:
		return "images/workspace-go"
	case ToolchainJava:
		return "images/workspace-java"
	case ToolchainNode:
		return "images/workspace-node"
	default:
		return ""
	}
}

func (f Toolchain) RecommendedExtensions() []string {
	switch f {
	case ToolchainPython:
		return []string{
			"ms-vscode-remote.remote-ssh",
			"ms-python.python",
			"ms-python.vscode-pylance",
			"charliermarsh.ruff",
		}
	case ToolchainGo:
		return []string{
			"ms-vscode-remote.remote-ssh",
			"golang.go",
		}
	case ToolchainJava:
		return []string{
			"ms-vscode-remote.remote-ssh",
			"vscjava.vscode-java-pack",
		}
	case ToolchainNode:
		return []string{remoteSSHExtension}
	default:
		return nil
	}
}

func (f Toolchain) Settings(mountDir string) map[string]any {
	switch f {
	case ToolchainPython:
		return map[string]any{
			"python.defaultInterpreterPath":            mountDir + "/.venv/bin/python",
			"python.terminal.activateEnvironment":      true,
			"terminal.integrated.defaultProfile.linux": "zsh",
		}
	case ToolchainGo, ToolchainNode:
		return map[string]any{
			"terminal.integrated.defaultProfile.linux": "zsh",
		}
	case ToolchainJava:
		return map[string]any{
			"terminal.integrated.defaultProfile.linux": "zsh",
		}
	default:
		return map[string]any{}
	}
}

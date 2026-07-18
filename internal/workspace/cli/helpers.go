package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/cofy-x/elyro/internal/workspace"
)

func expandSSHConfigPath(path string) (string, error) {
	expanded, err := workspace.ExpandPath(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(expanded)
}

func findElyroRepoRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	current := abs
	for {
		if fileExists(filepath.Join(current, "Makefile")) && fileExists(filepath.Join(current, "images", "workspace-base", "Dockerfile")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("elyro build is a source-checkout workflow; run it inside a cloned elyro repository")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func displayToolchain(toolchain string) string {
	if strings.TrimSpace(toolchain) == "" {
		return "none"
	}
	return toolchain
}

func displayEnvironment(environment, toolchain string) string {
	if strings.TrimSpace(environment) != "" {
		return environment
	}
	if strings.TrimSpace(toolchain) != "" {
		return toolchain
	}
	return "unknown"
}

func displayPlatform(platform string) string {
	if strings.TrimSpace(platform) == "" {
		return workspace.DefaultPlatform
	}
	return platform
}

func displayPrivileged(value string) string {
	if strings.TrimSpace(value) == "" {
		return "false"
	}
	return value
}

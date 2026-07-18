package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var toolchainMarkers = []struct {
	toolchain Toolchain
	markers   []string
}{
	{ToolchainPython, []string{"pyproject.toml", "requirements.txt", "Pipfile", "uv.lock", "setup.py"}},
	{ToolchainGo, []string{"go.mod", "go.work"}},
	{ToolchainJava, []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradlew"}},
	{ToolchainNode, []string{"package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock"}},
}

// ToolchainDetectionError reports that a project could not be mapped to exactly
// one built-in workspace toolchain.
type ToolchainDetectionError struct {
	ProjectDir string
	Matches    []Toolchain
}

func (e *ToolchainDetectionError) Error() string {
	if len(e.Matches) == 0 {
		return fmt.Sprintf("cannot detect a workspace toolchain for %s; pass --toolchain python, go, java, or node", e.ProjectDir)
	}
	names := make([]string, 0, len(e.Matches))
	for _, toolchain := range e.Matches {
		names = append(names, string(toolchain))
	}
	return fmt.Sprintf("multiple workspace toolchains detected for %s (%s); pass --toolchain python, go, java, or node", e.ProjectDir, strings.Join(names, ", "))
}

func DetectToolchains(projectDir string) ([]Toolchain, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolve project dir: %w", err)
	}

	matches := make([]Toolchain, 0, len(toolchainMarkers))
	for _, candidate := range toolchainMarkers {
		for _, marker := range candidate.markers {
			info, statErr := os.Stat(filepath.Join(absDir, marker))
			if statErr == nil && !info.IsDir() {
				matches = append(matches, candidate.toolchain)
				break
			}
			if statErr != nil && !os.IsNotExist(statErr) {
				return nil, fmt.Errorf("inspect project marker %s: %w", marker, statErr)
			}
		}
	}
	return matches, nil
}

func DetectToolchain(projectDir string) (Toolchain, error) {
	matches, err := DetectToolchains(projectDir)
	if err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", &ToolchainDetectionError{ProjectDir: projectDir, Matches: matches}
	}
	return matches[0], nil
}

package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ProjectRoot struct {
	Dir        string
	Source     ProjectRootSource
	ConfigPath string
}

type ProjectRootSource string

const (
	ProjectRootSourceExplicit ProjectRootSource = "explicit"
	ProjectRootSourceConfig   ProjectRootSource = "config"
	ProjectRootSourceRegistry ProjectRootSource = "registry"
	ProjectRootSourceGit      ProjectRootSource = "git"
	ProjectRootSourceCWD      ProjectRootSource = "cwd"
)

func ResolveProjectRoot(projectDir string, explicit bool) (ProjectRoot, error) {
	expanded, err := ExpandPath(projectDir)
	if err != nil {
		return ProjectRoot{}, err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return ProjectRoot{}, fmt.Errorf("resolve project dir: %w", err)
	}
	if explicit {
		return ProjectRoot{Dir: filepath.Clean(abs), Source: ProjectRootSourceExplicit}, nil
	}

	start := physicalPath(abs)
	if configDir, configPath, found, err := findAncestorEntry(start, defaultEnvironmentConfigFile); err != nil {
		return ProjectRoot{}, err
	} else if found {
		return ProjectRoot{Dir: configDir, Source: ProjectRootSourceConfig, ConfigPath: configPath}, nil
	}

	registryPath, err := DefaultPath()
	if err != nil {
		return ProjectRoot{}, err
	}
	store, err := Load(registryPath)
	if err != nil {
		return ProjectRoot{}, err
	}
	if record, err := Current(store, start); err == nil {
		return ProjectRoot{Dir: record.ProjectDir, Source: ProjectRootSourceRegistry}, nil
	} else if !errors.Is(err, ErrNoCurrent) {
		return ProjectRoot{}, err
	}

	if gitDir, _, found, err := findAncestorEntry(start, ".git"); err != nil {
		return ProjectRoot{}, err
	} else if found {
		return ProjectRoot{Dir: gitDir, Source: ProjectRootSourceGit}, nil
	}
	return ProjectRoot{Dir: start, Source: ProjectRootSourceCWD}, nil
}

func HasProjectSignal(projectDir string) (bool, error) {
	root, err := ResolveProjectRoot(projectDir, false)
	if err != nil {
		return false, err
	}
	if root.Source != ProjectRootSourceCWD {
		return true, nil
	}
	matches, err := DetectToolchains(root.Dir)
	if err != nil {
		return false, err
	}
	return len(matches) > 0, nil
}

func findAncestorEntry(start, name string) (string, string, bool, error) {
	for dir := filepath.Clean(start); ; dir = filepath.Dir(dir) {
		entryPath := filepath.Join(dir, name)
		if _, err := os.Lstat(entryPath); err == nil {
			return dir, entryPath, true, nil
		} else if !os.IsNotExist(err) {
			return "", "", false, fmt.Errorf("inspect project marker %s: %w", entryPath, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false, nil
		}
	}
}

package workspace

import (
	"fmt"
	"sort"
)

// ProjectImageConfig is the configuration subset needed by `elyro image init`.
type ProjectImageConfig struct {
	Path               string
	DefaultEnvironment string
	Environments       map[string]ProjectImageEnvironment
}

type ProjectImageEnvironment struct {
	Toolchain  string
	Image      string
	HasBuild   bool
	Context    string
	Dockerfile string
	Platform   string
}

func LoadProjectImageConfig(projectDir string) (*ProjectImageConfig, error) {
	cfg, path, err := loadProjectConfig(projectDir)
	if err != nil || cfg == nil {
		return nil, err
	}
	result := &ProjectImageConfig{
		Path: path, DefaultEnvironment: cfg.DefaultEnvironment,
		Environments: make(map[string]ProjectImageEnvironment, len(cfg.Environments)),
	}
	for name, environment := range cfg.Environments {
		result.Environments[name] = ProjectImageEnvironment{
			Toolchain: environment.Toolchain, Image: environment.Image,
			HasBuild: environment.Build.Set, Context: environment.Build.Context,
			Dockerfile: environment.Build.Dockerfile, Platform: environment.Platform,
		}
	}
	return result, nil
}

func ValidateProjectConfiguration(projectDir string) error {
	cfg, _, err := loadProjectConfig(projectDir)
	if err != nil || cfg == nil {
		return err
	}
	if cfg.DefaultEnvironment != "" {
		if _, ok := cfg.Environments[cfg.DefaultEnvironment]; !ok {
			return fmt.Errorf("default environment %q not found in elyro.yaml", cfg.DefaultEnvironment)
		}
	}
	if len(cfg.Environments) == 0 {
		return fmt.Errorf("elyro.yaml must define at least one Environment")
	}
	names := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := cfg.resolveNamedEnvironment(projectDir, name, "/home/elyro/project", EnvironmentSelection{}); err != nil {
			return err
		}
	}
	return nil
}

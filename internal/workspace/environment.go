package workspace

import (
	"errors"
	"fmt"

	elyroimages "github.com/cofy-x/elyro/internal/images"
)

const (
	remoteSSHExtension = "ms-vscode-remote.remote-ssh"
)

var DefaultPlatform = elyroimages.DefaultPlatform()

type EnvironmentSelection struct {
	Environment         string
	Toolchain           string
	Platform            string
	EnvironmentExplicit bool
	ToolchainExplicit   bool
	PlatformExplicit    bool
}

type ResolvedEnvironment struct {
	Name      string
	Toolchain Toolchain
	Image     string
	// CustomImage reports that elyro.yaml explicitly selected Image instead of deriving it from the toolchain.
	CustomImage           bool
	ProjectConfigured     bool
	Platform              string
	Docker                DockerOptions
	RecommendedExtensions []string
	Settings              map[string]any
}

func ResolveEnvironment(projectDir, mountDir string, selection EnvironmentSelection) (ResolvedEnvironment, error) {
	if selection.EnvironmentExplicit && selection.ToolchainExplicit {
		return ResolvedEnvironment{}, errors.New("--environment and --toolchain cannot be used together")
	}
	if selection.PlatformExplicit {
		if err := ValidatePlatform(selection.Platform); err != nil {
			return ResolvedEnvironment{}, err
		}
	}

	cfg, cfgPath, err := loadProjectConfig(projectDir)
	if err != nil {
		return ResolvedEnvironment{}, err
	}

	if selection.EnvironmentExplicit {
		if cfg == nil {
			return ResolvedEnvironment{}, fmt.Errorf("workspace config not found: %s", cfgPath)
		}
		return cfg.resolveNamedEnvironment(projectDir, selection.Environment, mountDir, selection)
	}

	if selection.ToolchainExplicit {
		toolchain, err := ParseToolchain(selection.Toolchain)
		if err != nil {
			return ResolvedEnvironment{}, err
		}
		return builtinResolvedEnvironment(toolchain, mountDir, selection.Platform), nil
	}

	if cfg != nil && cfg.DefaultEnvironment != "" {
		return cfg.resolveNamedEnvironment(projectDir, cfg.DefaultEnvironment, mountDir, selection)
	}

	detected, err := DetectToolchain(projectDir)
	if err != nil {
		return ResolvedEnvironment{}, err
	}
	return builtinResolvedEnvironment(detected, mountDir, selection.Platform), nil
}

func (cfg *projectConfig) resolveNamedEnvironment(projectDir, name, mountDir string, selection EnvironmentSelection) (ResolvedEnvironment, error) {
	if cfg == nil {
		return ResolvedEnvironment{}, errors.New("workspace config is required")
	}

	environment, ok := cfg.Environments[name]
	if !ok {
		return ResolvedEnvironment{}, fmt.Errorf("environment %q not found in %s", name, defaultEnvironmentConfigFile)
	}

	resolved := ResolvedEnvironment{
		Name:                  name,
		ProjectConfigured:     true,
		Platform:              DefaultPlatform,
		RecommendedExtensions: []string{remoteSSHExtension},
		Settings:              map[string]any{},
	}

	if environment.Toolchain != "" {
		toolchain, err := ParseToolchain(environment.Toolchain)
		if err != nil {
			return ResolvedEnvironment{}, err
		}
		resolved = builtinResolvedEnvironment(toolchain, mountDir, DefaultPlatform)
		resolved.Name = name
		resolved.ProjectConfigured = true
	}

	if environment.Image != "" {
		resolved.Image = environment.Image
		resolved.CustomImage = true
	}
	if resolved.Image == "" {
		return ResolvedEnvironment{}, fmt.Errorf("environment %q must set image or toolchain", name)
	}
	if environment.Platform != "" {
		if err := ValidatePlatform(environment.Platform); err != nil {
			return ResolvedEnvironment{}, fmt.Errorf("environment %q: %w", name, err)
		}
		resolved.Platform = environment.Platform
	}
	if selection.PlatformExplicit {
		resolved.Platform = selection.Platform
	}
	if resolved.Toolchain != "" && environment.Image == "" {
		resolved.Image = resolved.Toolchain.Image(resolved.Platform)
	}

	resolved.RecommendedExtensions = append(resolved.RecommendedExtensions, environment.VSCode.Extensions...)
	resolved.Settings = mergeSettings(resolved.Settings, environment.VSCode.Settings)
	dockerOptions, err := resolveDockerOptions(projectDir, environment.Docker)
	if err != nil {
		return ResolvedEnvironment{}, fmt.Errorf("environment %q: %w", name, err)
	}
	resolved.Docker = dockerOptions
	return resolved, nil
}

func builtinResolvedEnvironment(toolchain Toolchain, mountDir, platform string) ResolvedEnvironment {
	if platform == "" {
		platform = DefaultPlatform
	}
	return ResolvedEnvironment{
		Name:                  string(toolchain),
		Toolchain:             toolchain,
		Image:                 toolchain.Image(platform),
		Platform:              platform,
		Docker:                DockerOptions{},
		RecommendedExtensions: toolchain.RecommendedExtensions(),
		Settings:              toolchain.Settings(mountDir),
	}
}

func ValidatePlatform(platform string) error {
	switch platform {
	case "linux/amd64", "linux/arm64":
		return nil
	default:
		return fmt.Errorf("unsupported platform %q (supported: linux/amd64, linux/arm64)", platform)
	}
}

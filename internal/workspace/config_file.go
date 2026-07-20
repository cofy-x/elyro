package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultEnvironmentConfigFile = "elyro.yaml"

type projectConfig struct {
	Version            int                              `yaml:"version"`
	DefaultEnvironment string                           `yaml:"default_environment"`
	Environments       map[string]configuredEnvironment `yaml:"environments"`
}

type configuredEnvironment struct {
	Toolchain string           `yaml:"toolchain"`
	Image     string           `yaml:"image"`
	Build     configuredBuild  `yaml:"build,omitempty"`
	Platform  string           `yaml:"platform"`
	Docker    configuredDocker `yaml:"docker"`
	VSCode    configuredVSCode `yaml:"vscode"`
}

func (environment *configuredEnvironment) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("environment must be a mapping")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "build" && node.Content[i+1].Kind != yaml.MappingNode {
			return fmt.Errorf("build must be a mapping")
		}
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	type plainEnvironment configuredEnvironment
	var decoded plainEnvironment
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*environment = configuredEnvironment(decoded)
	return nil
}

type configuredBuild struct {
	Set        bool   `yaml:"-"`
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

func (build *configuredBuild) UnmarshalYAML(node *yaml.Node) error {
	build.Set = true
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("build must be a mapping")
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if seen[key] {
			return fmt.Errorf("build field %q is duplicated", key)
		}
		seen[key] = true
		switch key {
		case "context":
			if err := value.Decode(&build.Context); err != nil {
				return fmt.Errorf("build.context: %w", err)
			}
		case "dockerfile":
			if err := value.Decode(&build.Dockerfile); err != nil {
				return fmt.Errorf("build.dockerfile: %w", err)
			}
		default:
			return fmt.Errorf("unknown image build field %q", key)
		}
	}
	return nil
}

func loadProjectConfig(projectDir string) (*projectConfig, string, error) {
	configPath := filepath.Join(projectDir, defaultEnvironmentConfigFile)
	return loadConfigFile(configPath)
}

func loadConfigFile(configPath string) (*projectConfig, string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, configPath, nil
		}
		return nil, configPath, fmt.Errorf("read %s: %w", configPath, err)
	}

	var cfg projectConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, configPath, fmt.Errorf("parse %s: %w", configPath, err)
	}
	if cfg.Version != 1 {
		return nil, configPath, fmt.Errorf("%s: unsupported version %d", configPath, cfg.Version)
	}
	return &cfg, configPath, nil
}

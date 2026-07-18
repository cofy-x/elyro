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
	Platform  string           `yaml:"platform"`
	Docker    configuredDocker `yaml:"docker"`
	VSCode    configuredVSCode `yaml:"vscode"`
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

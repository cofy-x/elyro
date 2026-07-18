package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

func ProjectConfigPath(projectDir string) (string, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolve project dir: %w", err)
	}
	return filepath.Join(absDir, defaultEnvironmentConfigFile), nil
}

func WriteProjectConfig(projectDir string, toolchain Toolchain) (string, error) {
	if _, err := ParseToolchain(string(toolchain)); err != nil {
		return "", err
	}
	configPath, err := ValidateProjectConfigTarget(projectDir)
	if err != nil {
		return "", err
	}

	data := []byte(fmt.Sprintf("version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: %s\n", toolchain))
	temp, err := os.CreateTemp(filepath.Dir(configPath), ".elyro.yaml-*")
	if err != nil {
		return "", fmt.Errorf("create temporary workspace config: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return "", fmt.Errorf("set temporary workspace config permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return "", fmt.Errorf("write temporary workspace config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close temporary workspace config: %w", err)
	}
	if err := os.Link(tempPath, configPath); err != nil {
		return "", fmt.Errorf("install %s: %w", configPath, err)
	}
	return configPath, nil
}

func ValidateProjectConfigTarget(projectDir string) (string, error) {
	configPath, err := ProjectConfigPath(projectDir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		return "", fmt.Errorf("inspect project directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", filepath.Dir(configPath))
	}
	if _, err := os.Stat(configPath); err == nil {
		return "", fmt.Errorf("workspace config already exists: %s", configPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect %s: %w", configPath, err)
	}

	return configPath, nil
}

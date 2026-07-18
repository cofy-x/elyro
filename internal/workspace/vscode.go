package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func EnsureVSCodeWorkspace(projectDir string, environment ResolvedEnvironment) error {
	return EnsureVSCodeWorkspaceConfig(projectDir, environment.RecommendedExtensions, environment.Settings)
}

func EnsureVSCodeWorkspaceConfig(projectDir string, recommendations []string, settings map[string]any) error {
	vsDir := filepath.Join(projectDir, ".vscode")
	if err := os.MkdirAll(vsDir, 0o755); err != nil {
		return fmt.Errorf("create .vscode dir: %w", err)
	}
	if err := ensureExtensionsJSON(filepath.Join(vsDir, "extensions.json"), recommendations); err != nil {
		return err
	}
	if err := ensureSettingsJSON(filepath.Join(vsDir, "settings.json"), settings); err != nil {
		return err
	}
	return nil
}

func ensureExtensionsJSON(path string, recommendations []string) error {
	type extensionsFile struct {
		Recommendations []string `json:"recommendations"`
	}
	current := extensionsFile{}
	if err := loadJSON(path, &current); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(current.Recommendations)+len(recommendations))
	for _, item := range current.Recommendations {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}
	for _, item := range recommendations {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}
	sort.Strings(merged)
	current.Recommendations = merged
	return saveJSON(path, current)
}

func ensureSettingsJSON(path string, settings map[string]any) error {
	current := map[string]any{}
	if err := loadJSON(path, &current); err != nil {
		return err
	}
	for key, value := range settings {
		current[key] = value
	}
	return saveJSON(path, current)
}

func loadJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func saveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

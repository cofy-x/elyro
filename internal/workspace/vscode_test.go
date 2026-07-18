package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureVSCodeWorkspaceMergesExistingFiles(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	vsDir := filepath.Join(projectDir, ".vscode")
	if err := os.MkdirAll(vsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vsDir, "extensions.json"), []byte("{\"recommendations\":[\"existing.extension\"]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vsDir, "settings.json"), []byte("{\"editor.tabSize\":2}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	environment := builtinResolvedEnvironment(ToolchainPython, "/home/elyro/demo", DefaultPlatform)
	environment.RecommendedExtensions = append(environment.RecommendedExtensions, "redhat.vscode-yaml")
	environment.Settings["editor.formatOnSave"] = true

	if err := EnsureVSCodeWorkspace(projectDir, environment); err != nil {
		t.Fatalf("EnsureVSCodeWorkspace returned error: %v", err)
	}

	extensionsData, err := os.ReadFile(filepath.Join(vsDir, "extensions.json"))
	if err != nil {
		t.Fatal(err)
	}
	extensions := string(extensionsData)
	if !strings.Contains(extensions, "existing.extension") || !strings.Contains(extensions, "ms-python.python") || !strings.Contains(extensions, "redhat.vscode-yaml") {
		t.Fatalf("extensions were not merged as expected:\n%s", extensions)
	}

	settingsData, err := os.ReadFile(filepath.Join(vsDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := string(settingsData)
	if !strings.Contains(settings, "\"editor.tabSize\": 2") {
		t.Fatalf("existing settings were removed:\n%s", settings)
	}
	if !strings.Contains(settings, "\"python.defaultInterpreterPath\": \"/home/elyro/demo/.venv/bin/python\"") {
		t.Fatalf("python settings missing:\n%s", settings)
	}
	if !strings.Contains(settings, "\"editor.formatOnSave\": true") {
		t.Fatalf("custom settings missing:\n%s", settings)
	}
}

package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEnvironmentDetectsBuiltinToolchainWithoutConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{})
	if err != nil {
		t.Fatalf("ResolveEnvironment returned error: %v", err)
	}

	if got, want := environment.Name, "go"; got != want {
		t.Fatalf("environment.Name = %q, want %q", got, want)
	}
	if got, want := environment.Image, ToolchainGo.Image(DefaultPlatform); got != want {
		t.Fatalf("environment.Image = %q, want %q", got, want)
	}
	if got, want := environment.Toolchain, ToolchainGo; got != want {
		t.Fatalf("environment.Toolchain = %q, want %q", got, want)
	}
	if environment.CustomImage {
		t.Fatal("environment.CustomImage = true, want false for built-in toolchain image")
	}
	if environment.ProjectConfigured {
		t.Fatal("zero-config resolution unexpectedly marked the project configured")
	}
	if got, want := environment.Platform, DefaultPlatform; got != want {
		t.Fatalf("environment.Platform = %q, want %q", got, want)
	}
}

func TestResolveEnvironmentUsesDefaultConfiguredEnvironment(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
version: 1
default_environment: api
environments:
  api:
    toolchain: go
    image: example.com/team/api-workspace:latest
    platform: linux/arm64
    vscode:
      extensions:
        - redhat.vscode-yaml
      settings:
        go.toolsManagement.autoUpdate: true
`)

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{})
	if err != nil {
		t.Fatalf("ResolveEnvironment returned error: %v", err)
	}

	if got, want := environment.Name, "api"; got != want {
		t.Fatalf("environment.Name = %q, want %q", got, want)
	}
	if got, want := environment.Image, "example.com/team/api-workspace:latest"; got != want {
		t.Fatalf("environment.Image = %q, want %q", got, want)
	}
	if got, want := environment.Toolchain, ToolchainGo; got != want {
		t.Fatalf("environment.Toolchain = %q, want %q", got, want)
	}
	if !environment.CustomImage {
		t.Fatal("environment.CustomImage = false, want true for configured image")
	}
	if !environment.ProjectConfigured {
		t.Fatal("configured Environment was not marked project-configured")
	}
	if got, want := environment.Platform, "linux/arm64"; got != want {
		t.Fatalf("environment.Platform = %q, want %q", got, want)
	}
	if got := environment.Docker.Privileged; got {
		t.Fatalf("environment.Docker.Privileged = %t, want false", got)
	}
	if !contains(environment.RecommendedExtensions, remoteSSHExtension) {
		t.Fatalf("expected remote SSH recommendation in %#v", environment.RecommendedExtensions)
	}
	if !contains(environment.RecommendedExtensions, "redhat.vscode-yaml") {
		t.Fatalf("expected custom recommendation in %#v", environment.RecommendedExtensions)
	}
	if got := environment.Settings["go.toolsManagement.autoUpdate"]; got != true {
		t.Fatalf("custom setting missing, got %#v", got)
	}
	if got := environment.Settings["terminal.integrated.defaultProfile.linux"]; got != "zsh" {
		t.Fatalf("builtin setting missing, got %#v", got)
	}
}

func TestResolveEnvironmentExplicitToolchainOverridesDefaultConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
version: 1
default_environment: api
environments:
  api:
    toolchain: go
`)

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Toolchain:         "python",
		ToolchainExplicit: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if environment.Name != "python" || environment.Toolchain != ToolchainPython {
		t.Fatalf("environment = %#v, want explicit python toolchain", environment)
	}
	if environment.ProjectConfigured {
		t.Fatal("explicit toolchain unexpectedly selected the configured Environment")
	}
}

func TestResolveEnvironmentExplicitEnvironmentOverridesDefault(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
version: 1
default_environment: api
environments:
  api:
    toolchain: go
  worker:
    image: example.com/team/worker:latest
`)

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "worker",
		EnvironmentExplicit: true,
	})
	if err != nil {
		t.Fatalf("ResolveEnvironment returned error: %v", err)
	}

	if got, want := environment.Name, "worker"; got != want {
		t.Fatalf("environment.Name = %q, want %q", got, want)
	}
	if got, want := environment.Image, "example.com/team/worker:latest"; got != want {
		t.Fatalf("environment.Image = %q, want %q", got, want)
	}
	if got := environment.Toolchain; got != "" {
		t.Fatalf("environment.Toolchain = %q, want empty", got)
	}
	if !environment.CustomImage {
		t.Fatal("environment.CustomImage = false, want true for image-only environment")
	}
	if got, want := environment.Platform, DefaultPlatform; got != want {
		t.Fatalf("environment.Platform = %q, want %q", got, want)
	}
}

func TestResolveEnvironmentPlatformFlagOverridesConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
environments:
  api:
    toolchain: go
    platform: linux/arm64
`)

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "api",
		Platform:            "linux/amd64",
		EnvironmentExplicit: true,
		PlatformExplicit:    true,
	})
	if err != nil {
		t.Fatalf("ResolveEnvironment returned error: %v", err)
	}
	if got, want := environment.Platform, "linux/amd64"; got != want {
		t.Fatalf("environment.Platform = %q, want %q", got, want)
	}
}

func writeEnvironmentConfig(t *testing.T, projectDir, content string) {
	t.Helper()

	path := filepath.Join(projectDir, defaultEnvironmentConfigFile)
	data := strings.TrimLeft(content, "\n")
	if !strings.HasPrefix(data, "version:") {
		data = "version: 1\n" + data
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

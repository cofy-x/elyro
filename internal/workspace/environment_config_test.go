package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEnvironmentRejectsUnknownFields(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), []byte("version: 1\nenvironments: {}\nunexpected: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{})
	if err == nil || !strings.Contains(err.Error(), "field unexpected not found") {
		t.Fatalf("ResolveEnvironment() error = %v, want strict unknown-field rejection", err)
	}
}

func TestResolveEnvironmentRejectsEnvironmentAndToolchainTogether(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "api",
		Toolchain:           "go",
		EnvironmentExplicit: true,
		ToolchainExplicit:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "--environment and --toolchain cannot be used together") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func TestResolveEnvironmentRequiresConfigForExplicitEnvironment(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "api",
		EnvironmentExplicit: true,
	})
	if err == nil || !strings.Contains(err.Error(), "workspace config not found") {
		t.Fatalf("expected config missing error, got %v", err)
	}
}

func TestResolveEnvironmentRequiresImageOrToolchain(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
environments:
  broken:
    vscode:
      extensions:
        - golang.go
`)

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "broken",
		EnvironmentExplicit: true,
	})
	if err == nil || !strings.Contains(err.Error(), "must set image or toolchain") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestResolveEnvironmentRejectsUnknownToolchain(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
environments:
  api:
    toolchain: rust
`)

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "api",
		EnvironmentExplicit: true,
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported toolchain "rust"`) {
		t.Fatalf("expected unknown toolchain error, got %v", err)
	}
}

func TestResolveEnvironmentRejectsUnknownPlatform(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Platform:         "linux/s390x",
		PlatformExplicit: true,
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported platform "linux/s390x"`) {
		t.Fatalf("expected platform validation error, got %v", err)
	}
}

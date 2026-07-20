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

func TestResolveEnvironmentStrictlyParsesImageBuild(t *testing.T) {
	t.Parallel()
	for _, build := range []string{
		"build: null\n",
		"build:\n      context: .\n      dockerfile: Dockerfile\n      unexpected: true\n",
		"build:\n      context: .\n      context: other\n      dockerfile: Dockerfile\n",
	} {
		projectDir := t.TempDir()
		config := "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    image: elyro-local/demo:dev\n    " + build
		if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{}); err == nil {
			t.Fatalf("ResolveEnvironment accepted invalid build:\n%s", config)
		}
	}
}

func TestResolveEnvironmentRequiresImageForBuild(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeEnvironmentConfig(t, projectDir, `
default_environment: dev
environments:
  dev:
    toolchain: go
    build:
      context: .
      dockerfile: Dockerfile
`)
	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{})
	if err == nil || !strings.Contains(err.Error(), "build requires image") {
		t.Fatalf("error = %v", err)
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

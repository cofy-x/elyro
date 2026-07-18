package images

import (
	"testing"

	elyroversion "github.com/cofy-x/elyro/internal/version"
)

func TestReferenceUsesReleaseManifestTag(t *testing.T) {
	previous := elyroversion.Version
	elyroversion.Version = "v0.1.0"
	t.Cleanup(func() { elyroversion.Version = previous })
	t.Setenv("ELYRO_IMAGE_PREFIX", "registry.example/elyro")
	if got, want := Reference("elyro/workspace-go", "linux/arm64"), "registry.example/elyro/workspace-go:v0.1.0"; got != want {
		t.Fatalf("Reference() = %q, want %q", got, want)
	}
}

func TestReferenceUsesArchitectureForDevelopment(t *testing.T) {
	previous := elyroversion.Version
	elyroversion.Version = "dev"
	t.Cleanup(func() { elyroversion.Version = previous })
	if got, want := Reference("elyro/workspace-go", "linux/amd64"), "ghcr.io/cofy-x/elyro/workspace-go:dev-amd64"; got != want {
		t.Fatalf("Reference() = %q, want %q", got, want)
	}
}

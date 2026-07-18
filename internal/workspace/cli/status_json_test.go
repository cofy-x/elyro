package cli

import (
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestLocalStatusPayloadWithoutContainer(t *testing.T) {
	t.Parallel()

	payload := localStatusPayload(local.StatusResult{
		Project: workspace.ProjectContext{
			ProjectDir:  "/tmp/demo",
			Slug:        "demo",
			ProjectHash: "deadbeef",
			MountDir:    "/home/elyro/demo",
		},
	})
	if payload.Kind != "workspace" {
		t.Fatalf("Kind = %q, want workspace", payload.Kind)
	}
	if payload.SchemaVersion != 1 || payload.Workspace.ID != "deadbeef" {
		t.Fatalf("payload = %#v, want schema 1 workspace", payload)
	}
	if payload.Workspace.Status != "absent" {
		t.Fatalf("Status = %q, want absent", payload.Workspace.Status)
	}
}

func TestLocalStatusPayloadWithContainer(t *testing.T) {
	t.Parallel()

	payload := localStatusPayload(local.StatusResult{
		Project: workspace.ProjectContext{
			ProjectDir:  "/tmp/demo",
			Slug:        "demo",
			ProjectHash: "deadbeef",
			MountDir:    "/home/elyro/demo",
		},
		Container: &dockerruntime.Container{
			ID:         "abc123",
			Name:       "elyro-demo",
			Status:     "running",
			Hostname:   "demo",
			Toolchain:  "python",
			Platform:   "linux/arm64",
			Image:      "elyro/workspace-python:latest-arm64",
			HostAlias:  "elyro-demo",
			HostPort:   "39222",
			Privileged: "true",
			Published:  "18080:8000",
			Mounts:     "/tmp/cache:/cache:ro",
		},
	})
	if payload.Workspace.Toolchain != "python" {
		t.Fatalf("Toolchain = %q, want python", payload.Workspace.Toolchain)
	}
	if payload.Workspace.Hostname != "demo" {
		t.Fatalf("Hostname = %q, want demo", payload.Workspace.Hostname)
	}
	if len(payload.Workspace.PublishedPorts) != 1 || payload.Workspace.PublishedPorts[0] != "18080:8000" {
		t.Fatalf("PublishedPorts = %q, want 18080:8000", payload.Workspace.PublishedPorts)
	}
}

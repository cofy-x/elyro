package cli

import (
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestWorkspaceRecord(t *testing.T) {
	record := workspaceRecord(local.UpResult{
		Project: workspace.ProjectContext{
			ProjectDir: "/tmp/demo",
			Slug:       "demo",
			MountDir:   "/home/elyro/demo",
		},
		Environment: workspace.ResolvedEnvironment{
			Name:      "python",
			Toolchain: workspace.ToolchainPython,
			Platform:  "linux/amd64",
		},
		Container: dockerruntime.Container{
			Name:      "elyro-workspace-demo",
			Hostname:  "demo",
			HostAlias: "elyro-demo",
		},
	})
	if record.Name != "demo" || record.Kind != workspace.KindWorkspace {
		t.Fatalf("record identity = %#v, want demo workspace", record)
	}
	if record.HostWorkspaceDir != "/tmp/demo" || record.ContainerWorkspaceDir != "/home/elyro/demo" {
		t.Fatalf("record workspace dirs = %#v", record)
	}
	if record.Hostname != "demo" {
		t.Fatalf("record hostname = %q, want demo", record.Hostname)
	}
}

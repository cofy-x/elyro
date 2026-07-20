package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestDownPlanPayloadDescribesRemovalAndPreservation(t *testing.T) {
	plan := local.DownPlan{
		Project:   workspace.ProjectContext{ProjectDir: "/tmp/demo", Slug: "demo", ProjectHash: "deadbeef", MountDir: "/home/elyro/demo"},
		Container: &dockerruntime.Container{Status: "running", Environment: "dev", Toolchain: "go", Image: "example:dev", Platform: "linux/arm64"},
		Action:    local.WorkspacePlanActionRemove,
		Removes: []local.WorkspaceManagedResource{
			local.WorkspaceManagedResourceContainerWritableLayer,
			local.WorkspaceManagedResourceManagedSSH,
			local.WorkspaceManagedResourceKnownHosts,
			local.WorkspaceManagedResourceRegistryRecord,
		},
		Preserves: []local.WorkspacePreservedResource{
			local.WorkspacePreservedResourceProjectFiles,
			local.WorkspacePreservedResourceMountedHostData,
			local.WorkspacePreservedResourceLocalImages,
		},
	}
	data, err := json.Marshal(downPlanPayload(plan, workspace.ProjectRoot{Dir: "/tmp/demo", Source: workspace.ProjectRootSourceExplicit}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`"kind":"workspace_plan"`, `"operation":"down"`, `"action":"remove"`, `"container_writable_layer"`, `"project_files"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("down plan missing %s: %s", want, text)
		}
	}
}

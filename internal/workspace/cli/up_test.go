package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestDisplayUpEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		environment workspace.ResolvedEnvironment
		want        string
	}{
		{name: "named environment", environment: workspace.ResolvedEnvironment{Name: "dev", Toolchain: workspace.ToolchainGo}, want: "dev"},
		{name: "detected toolchain", environment: workspace.ResolvedEnvironment{Toolchain: workspace.ToolchainGo}, want: "go"},
		{name: "custom environment", environment: workspace.ResolvedEnvironment{Name: "api"}, want: "api"},
		{name: "unnamed custom image", environment: workspace.ResolvedEnvironment{}, want: "custom"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := displayUpEnvironment(test.environment); got != test.want {
				t.Fatalf("displayUpEnvironment() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDisplayUpAction(t *testing.T) {
	t.Parallel()

	for action, want := range map[string]string{
		"created": "created",
		"started": "started",
		"reused":  "reused",
		"":        "ready",
		"unknown": "ready",
	} {
		if got := displayUpAction(action); got != want {
			t.Fatalf("displayUpAction(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestUpPayloadUsesWorkspaceSchema(t *testing.T) {
	view := upPayload(local.UpResult{
		Project:   workspace.ProjectContext{ProjectDir: "/tmp/demo", Slug: "demo", ProjectHash: "deadbeef", MountDir: "/home/elyro/demo"},
		Container: dockerruntime.Container{ID: "abc", Name: "elyro-workspace-demo", Status: "running", Hostname: "demo", Toolchain: "go", Platform: "linux/arm64"},
		Action:    "created",
	}, 1250*time.Millisecond)
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["schema_version"] != float64(1) || got["action"] != "created" || got["duration_ms"] != float64(1250) {
		t.Fatalf("up payload = %s", data)
	}
}

package cli

import (
	"encoding/json"
	"strings"
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

	for action, want := range map[local.WorkspaceAction]string{
		local.WorkspaceActionCreated:     "created",
		local.WorkspaceActionRecreated:   "recreated",
		local.WorkspaceActionStarted:     "started",
		local.WorkspaceActionReused:      "reused",
		local.WorkspaceAction(""):        "ready",
		local.WorkspaceAction("unknown"): "ready",
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
		Action:    local.WorkspaceActionCreated,
		Reasons:   []local.WorkspaceChangeReason{local.WorkspaceChangeReasonAbsent},
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
	reasons, ok := got["reasons"].([]any)
	if !ok || len(reasons) != 1 || reasons[0] != "workspace_absent" {
		t.Fatalf("up payload reasons = %#v", got["reasons"])
	}
}

func TestUpPlanPayloadIsStableAndRedactsRuntimeValues(t *testing.T) {
	const sentinel = "runtime-secret-sentinel"
	plan := local.UpPlan{
		Project: workspace.ProjectContext{ProjectDir: "/tmp/demo", Slug: "demo", ProjectHash: "deadbeef", MountDir: "/home/elyro/demo"},
		Environment: workspace.ResolvedEnvironment{
			Name: "dev", Toolchain: workspace.ToolchainGo, Image: "elyro/workspace-go:v0.1.5", Platform: "linux/arm64",
			Docker: workspace.DockerOptions{RuntimeEnvironment: workspace.RuntimeEnvironment{Effective: map[string]string{"SENTINEL": sentinel}, Digest: "sha256:secret"}},
		},
		Action: local.WorkspacePlanActionRecreate, Reasons: []local.WorkspaceChangeReason{local.WorkspaceChangeReasonRuntimeEnvironment},
		ImageStatus: local.WorkspaceImageStatusAvailable,
		Container:   &dockerruntime.Container{Status: "running"},
	}
	data, err := json.Marshal(upPlanPayload(plan, workspace.ProjectRoot{Dir: "/tmp/demo", Source: workspace.ProjectRootSourceConfig}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{sentinel, "sha256:secret", "SENTINEL"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("plan payload leaked %q: %s", forbidden, text)
		}
	}
	for _, want := range []string{`"kind":"workspace_plan"`, `"operation":"up"`, `"action":"recreate"`, `"runtime_environment_changed"`, `"source":"config"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("plan payload missing %s: %s", want, text)
		}
	}
}

func TestUpCommandForPlanPreservesExplicitInputs(t *testing.T) {
	request := local.UpRequest{
		ProjectDir: "/tmp/project with spaces", Environment: "dev", EnvironmentExplicit: true,
		Platform: "linux/arm64", PlatformExplicit: true, PublishSpecs: []string{"18080:8080"},
		AllowUnsafeEnvironment: true, Recreate: true,
	}
	want := "elyro up --project-dir '/tmp/project with spaces' --environment dev --platform linux/arm64 --publish 18080:8080 --allow-unsafe-environment --recreate"
	if got := upCommandForPlan(request, true); got != want {
		t.Fatalf("upCommandForPlan() = %q, want %q", got, want)
	}
	if got := quoteCommandArg("it's here"); got != `'it'\''s here'` {
		t.Fatalf("quoteCommandArg() = %q", got)
	}
}

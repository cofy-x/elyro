package local

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestPlanContainerActionCoversLifecycle(t *testing.T) {
	t.Parallel()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	matching := testContainer(project, environment, "running")
	stopped := testContainer(project, environment, "exited")
	tests := []struct {
		name     string
		info     *dockerruntime.Container
		recreate bool
		action   WorkspacePlanAction
		reasons  []WorkspaceChangeReason
	}{
		{name: "absent", action: WorkspacePlanActionCreate, reasons: []WorkspaceChangeReason{WorkspaceChangeReasonAbsent}},
		{name: "running", info: matching, action: WorkspacePlanActionReuse, reasons: []WorkspaceChangeReason{}},
		{name: "stopped", info: stopped, action: WorkspacePlanActionStart, reasons: []WorkspaceChangeReason{WorkspaceChangeReasonStopped}},
		{name: "explicit recreate", info: matching, recreate: true, action: WorkspacePlanActionRecreate, reasons: []WorkspaceChangeReason{WorkspaceChangeReasonExplicitRecreate}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			action, reasons := planContainerAction(test.info, project, environment, "", "", "false", test.recreate)
			if action != test.action || !slices.Equal(reasons, test.reasons) {
				t.Fatalf("plan = %q %#v, want %q %#v", action, reasons, test.action, test.reasons)
			}
		})
	}
}

func TestCompareContainerSpecificationReturnsStableReasonOrder(t *testing.T) {
	t.Parallel()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	environment.Docker.RuntimeEnvironment.Digest = "new-runtime"
	info := testContainer(project, environment, "running")
	info.Environment = "old"
	info.ImageLabel = "old-image"
	info.Platform = "linux/arm64"
	info.Hostname = "old-hostname"
	info.Published = "9000:9000"
	info.Mounts = "/old:/old"
	info.Privileged = "true"
	info.RuntimeEnvironmentDigest = "old-runtime"
	want := []WorkspaceChangeReason{
		WorkspaceChangeReasonEnvironment,
		WorkspaceChangeReasonImage,
		WorkspaceChangeReasonPlatform,
		WorkspaceChangeReasonHostname,
		WorkspaceChangeReasonPublishedPorts,
		WorkspaceChangeReasonMounts,
		WorkspaceChangeReasonPrivileged,
		WorkspaceChangeReasonRuntimeEnvironment,
	}
	got := CompareContainerSpecification(info, project, environment, "", "", "false")
	if !slices.Equal(got, want) {
		t.Fatalf("reasons = %#v, want %#v", got, want)
	}
}

func TestPlanUpHasNoRuntimeMutations(t *testing.T) {
	projectDir := t.TempDir()
	runtime := &fakeContainerRuntime{imageExists: true}
	plan, err := planUp(t.Context(), runtime, UpRequest{
		ProjectDir: projectDir, SSHConfigPath: filepath.Join(t.TempDir(), "config"),
		Toolchain: "go", ToolchainExplicit: true, Platform: workspace.DefaultPlatform,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != WorkspacePlanActionCreate || plan.ImageStatus != WorkspaceImageStatusAvailable {
		t.Fatalf("plan = %#v", plan)
	}
	if len(runtime.pulls) != 0 || len(runtime.runs) != 0 || len(runtime.starts) != 0 || len(runtime.removes) != 0 || len(runtime.waits) != 0 {
		t.Fatalf("dry-run mutated runtime: %#v", runtime)
	}
}

func TestPlanUpReportsOfficialImagePullWithoutPulling(t *testing.T) {
	runtime := &fakeContainerRuntime{imageExists: false}
	plan, err := planUp(t.Context(), runtime, UpRequest{
		ProjectDir: t.TempDir(), SSHConfigPath: filepath.Join(t.TempDir(), "config"),
		Toolchain: "go", ToolchainExplicit: true, Platform: workspace.DefaultPlatform,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ImageStatus != WorkspaceImageStatusPullRequired || len(runtime.pulls) != 0 {
		t.Fatalf("plan image status = %q, pulls = %#v", plan.ImageStatus, runtime.pulls)
	}
}

func TestPlanDownActionsAndResources(t *testing.T) {
	t.Parallel()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	tests := []struct {
		name      string
		container bool
		ssh       bool
		known     bool
		registry  bool
		action    WorkspacePlanAction
		removes   []WorkspaceManagedResource
	}{
		{name: "none", action: WorkspacePlanActionNone, removes: []WorkspaceManagedResource{}},
		{name: "cleanup", ssh: true, registry: true, action: WorkspacePlanActionCleanup, removes: []WorkspaceManagedResource{WorkspaceManagedResourceManagedSSH, WorkspaceManagedResourceRegistryRecord}},
		{name: "remove", container: true, ssh: true, known: true, registry: true, action: WorkspacePlanActionRemove, removes: []WorkspaceManagedResource{WorkspaceManagedResourceContainerWritableLayer, WorkspaceManagedResourceManagedSSH, WorkspaceManagedResourceKnownHosts, WorkspaceManagedResourceRegistryRecord}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := &fakeContainerRuntime{}
			if test.container {
				runtime.byProject = testContainer(project, environment, "running")
			}
			plan, err := planDown(t.Context(), runtime, DownPlanRequest{
				DownRequest: DownRequest{ProjectDir: project.ProjectDir}, ManagedSSHPresent: test.ssh,
				KnownHostPresent: test.known, RegistryPresent: test.registry,
			})
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != test.action || !slices.Equal(plan.Removes, test.removes) {
				t.Fatalf("plan = %q %#v, want %q %#v", plan.Action, plan.Removes, test.action, test.removes)
			}
			if len(runtime.removes) != 0 {
				t.Fatalf("plan removed container: %#v", runtime.removes)
			}
		})
	}
}

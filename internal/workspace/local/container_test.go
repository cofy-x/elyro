package local

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestEnsureContainerReusesRunningContainer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	runtime := &fakeContainerRuntime{
		byProject: testContainer(project, environment, "running"),
	}

	info, action, err := ensureTestContainer(ctx, runtime, project, environment, nil, "", "", "false", "", false)
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != WorkspaceActionReused {
		t.Fatalf("ensureContainer() action = %q, want reused", action)
	}
	if info.Name != project.ContainerName {
		t.Fatalf("ensureContainer() name = %q, want %q", info.Name, project.ContainerName)
	}
	if len(runtime.runs) != 0 || len(runtime.starts) != 0 || len(runtime.removes) != 0 {
		t.Fatalf("runtime mutated container: runs=%v starts=%v removes=%v", runtime.runs, runtime.starts, runtime.removes)
	}
}

func TestEnsureContainerStartsStoppedContainer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	runtime := &fakeContainerRuntime{
		byProject: testContainer(project, environment, "exited"),
		inspect: map[string]*dockerruntime.Container{
			project.ContainerName: testContainer(project, environment, "running"),
		},
	}

	info, action, err := ensureTestContainer(ctx, runtime, project, environment, nil, "", "", "false", "", false)
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != WorkspaceActionStarted {
		t.Fatalf("ensureContainer() action = %q, want started", action)
	}
	if info.Status != "running" {
		t.Fatalf("ensureContainer() status = %q, want running", info.Status)
	}
	if !slices.Equal(runtime.starts, []string{project.ContainerName}) {
		t.Fatalf("starts = %v, want %v", runtime.starts, []string{project.ContainerName})
	}
}

func TestEnsureContainerRebuildsMismatchedContainer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	oldContainer := testContainer(project, environment, "running")
	oldContainer.ImageLabel = "elyro/workspace-python:old"
	runtime := &fakeContainerRuntime{
		byProject: oldContainer,
		inspect: map[string]*dockerruntime.Container{
			project.ContainerName: testContainer(project, environment, "running"),
		},
	}

	_, action, err := ensureTestContainer(ctx, runtime, project, environment, nil, "", "", "false", "", false)
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != WorkspaceActionRecreated {
		t.Fatalf("ensureContainer() action = %q, want recreated", action)
	}
	if !slices.Equal(runtime.removes, []string{project.ContainerName}) {
		t.Fatalf("removes = %v, want %v", runtime.removes, []string{project.ContainerName})
	}
	if len(runtime.runs) != 1 {
		t.Fatalf("runs = %v, want one docker run", runtime.runs)
	}
}

func TestEnsureContainerRecreatesMatchingContainerWhenRequested(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	runtime := &fakeContainerRuntime{
		byProject: testContainer(project, environment, "running"),
		inspect: map[string]*dockerruntime.Container{
			project.ContainerName: testContainer(project, environment, "running"),
		},
	}

	_, action, err := ensureTestContainer(ctx, runtime, project, environment, nil, "", "", "false", "", true)
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != WorkspaceActionRecreated {
		t.Fatalf("ensureContainer() action = %q, want recreated", action)
	}
	if !slices.Equal(runtime.removes, []string{project.ContainerName}) || len(runtime.runs) != 1 {
		t.Fatalf("recreate mutations: removes=%v runs=%v", runtime.removes, runtime.runs)
	}
}

func TestEnsureContainerRecreateActionCoversExistingAndAbsentWorkspaces(t *testing.T) {
	t.Parallel()

	project := testProjectContext()
	environment := testResolvedEnvironment()
	for _, test := range []struct {
		name       string
		existing   *dockerruntime.Container
		wantAction WorkspaceAction
		wantRemove bool
	}{
		{name: "stopped", existing: testContainer(project, environment, "exited"), wantAction: WorkspaceActionRecreated, wantRemove: true},
		{name: "mismatched", existing: func() *dockerruntime.Container {
			container := testContainer(project, environment, "running")
			container.ImageLabel = "elyro/workspace-python:old"
			return container
		}(), wantAction: WorkspaceActionRecreated, wantRemove: true},
		{name: "absent", wantAction: WorkspaceActionCreated},
	} {
		t.Run(test.name, func(t *testing.T) {
			runtime := &fakeContainerRuntime{
				byProject: test.existing,
				inspect: map[string]*dockerruntime.Container{
					project.ContainerName: testContainer(project, environment, "running"),
				},
			}
			_, action, err := ensureTestContainer(t.Context(), runtime, project, environment, nil, "", "", "false", "", true)
			if err != nil {
				t.Fatal(err)
			}
			if action != test.wantAction {
				t.Fatalf("action = %q, want %q", action, test.wantAction)
			}
			if got := len(runtime.removes) == 1; got != test.wantRemove {
				t.Fatalf("removed existing container = %t, want %t", got, test.wantRemove)
			}
		})
	}
}

func TestEnsureContainerRejectsContainerNameOwnedByOtherProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := testProjectContext()
	environment := testResolvedEnvironment()
	runtime := &fakeContainerRuntime{
		byName: &dockerruntime.Container{
			Name:       project.ContainerName,
			ProjectDir: "/tmp/other",
		},
	}

	_, _, err := ensureTestContainer(ctx, runtime, project, environment, nil, "", "", "false", "", false)
	if err == nil {
		t.Fatal("ensureContainer() error = nil, want conflict")
	}
	if len(runtime.runs) != 0 {
		t.Fatalf("runs = %v, want none", runtime.runs)
	}
}

func ensureTestContainer(ctx context.Context, runtime containerRuntime, project workspace.ProjectContext, environment workspace.ResolvedEnvironment, publishes []workspace.PortPublish, normalizedPublishes, normalizedMounts, privilegedLabel, sshPort string, recreate bool) (*dockerruntime.Container, WorkspaceAction, error) {
	info, err := runtime.InspectByProject(ctx, project.ProjectDir)
	if err != nil {
		return nil, "", err
	}
	if info == nil {
		occupied, inspectErr := runtime.InspectByName(ctx, project.ContainerName)
		if inspectErr != nil {
			return nil, "", inspectErr
		}
		if occupied != nil {
			return nil, "", fmt.Errorf("container name conflict")
		}
	}
	action, reasons := planContainerAction(info, project, environment, normalizedPublishes, normalizedMounts, privilegedLabel, recreate)
	return executeContainerPlan(ctx, runtime, UpPlan{
		Project:         project,
		Environment:     environment,
		Container:       info,
		Action:          action,
		Reasons:         reasons,
		Publishes:       publishes,
		Published:       normalizedPublishes,
		Mounts:          normalizedMounts,
		PrivilegedLabel: privilegedLabel,
	}, sshPort)
}

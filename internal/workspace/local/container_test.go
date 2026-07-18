package local

import (
	"context"
	"slices"
	"testing"

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

	info, action, err := ensureContainer(ctx, runtime, project, environment, nil, "", "", "false", "")
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != "reused" {
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

	info, action, err := ensureContainer(ctx, runtime, project, environment, nil, "", "", "false", "")
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != "started" {
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

	_, action, err := ensureContainer(ctx, runtime, project, environment, nil, "", "", "false", "")
	if err != nil {
		t.Fatalf("ensureContainer() error = %v", err)
	}
	if action != "created" {
		t.Fatalf("ensureContainer() action = %q, want created", action)
	}
	if !slices.Equal(runtime.removes, []string{project.ContainerName}) {
		t.Fatalf("removes = %v, want %v", runtime.removes, []string{project.ContainerName})
	}
	if len(runtime.runs) != 1 {
		t.Fatalf("runs = %v, want one docker run", runtime.runs)
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

	_, _, err := ensureContainer(ctx, runtime, project, environment, nil, "", "", "false", "")
	if err == nil {
		t.Fatal("ensureContainer() error = nil, want conflict")
	}
	if len(runtime.runs) != 0 {
		t.Fatalf("runs = %v, want none", runtime.runs)
	}
}

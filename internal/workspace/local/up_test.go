package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestUpMissingBuiltinImageSuggestsBuildOrPull(t *testing.T) {
	_, err := up(t.Context(), &fakeContainerRuntime{imageExists: false, pullErr: errors.New("registry unavailable")}, UpRequest{
		ProjectDir:        t.TempDir(),
		SSHConfigPath:     "/tmp/ssh-config",
		Toolchain:         "python",
		ToolchainExplicit: true,
		Platform:          workspace.DefaultPlatform,
	})
	if err == nil {
		t.Fatal("up() error = nil, want missing image error")
	}
	if !strings.Contains(err.Error(), "build or pull") {
		t.Fatalf("up() error = %v, want build or pull hint", err)
	}
}

func TestUpMissingCustomImageSuggestsBuildOrPullWithoutPulling(t *testing.T) {
	tests := []struct {
		name      string
		toolchain string
	}{
		{name: "image only"},
		{name: "image with toolchain metadata", toolchain: "    toolchain: go\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := t.TempDir()
			config := []byte("version: 1\nenvironments:\n  custom:\n" + tt.toolchain + "    image: example/custom-workspace:local\n")
			if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), config, 0o644); err != nil {
				t.Fatal(err)
			}
			runtime := &fakeContainerRuntime{imageExists: false, byProject: &dockerruntime.Container{Name: "existing-workspace"}}

			_, err := up(t.Context(), runtime, UpRequest{
				ProjectDir:          projectDir,
				SSHConfigPath:       "/tmp/ssh-config",
				Environment:         "custom",
				EnvironmentExplicit: true,
				Recreate:            true,
			})
			if err == nil || !strings.Contains(err.Error(), "build or pull it first") {
				t.Fatalf("up() error = %v, want custom image build-or-pull hint", err)
			}
			if len(runtime.pulls) != 0 {
				t.Fatalf("custom image pulls = %#v, want none", runtime.pulls)
			}
			if len(runtime.removes) != 0 {
				t.Fatalf("preflight failure removed existing workspace: %#v", runtime.removes)
			}
		})
	}
}

func TestUpMissingProjectBuildImageSuggestsImageBuildWithoutTouchingWorkspace(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectDir, ".elyro"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".elyro", "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := []byte("version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    image: elyro-local/demo:dev\n    build:\n      context: .\n      dockerfile: .elyro/Dockerfile\n")
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), config, 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := &fakeContainerRuntime{byProject: &dockerruntime.Container{Name: "existing-workspace"}}
	_, err := up(t.Context(), runtime, UpRequest{ProjectDir: projectDir, SSHConfigPath: "/tmp/ssh-config", Recreate: true})
	if err == nil || !strings.Contains(err.Error(), "elyro image build") {
		t.Fatalf("up() error = %v, want image build hint", err)
	}
	if len(runtime.pulls) != 0 || len(runtime.removes) != 0 {
		t.Fatalf("preflight changed state: pulls=%#v removes=%#v", runtime.pulls, runtime.removes)
	}
}

func TestUpInvalidRuntimeEnvironmentPreservesExistingWorkspace(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectDir, ".elyro"), 0o755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "runtime-secret-sentinel"
	if err := os.WriteFile(filepath.Join(projectDir, ".elyro", "dev.env"), []byte("VALUE="+sentinel+"\nBROKEN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := []byte("version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    docker:\n      env_files:\n        - .elyro/dev.env\n")
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), config, 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := &fakeContainerRuntime{imageExists: true, byProject: &dockerruntime.Container{Name: "existing-workspace"}}
	_, err := up(t.Context(), runtime, UpRequest{ProjectDir: projectDir, SSHConfigPath: "/tmp/ssh-config", Recreate: true})
	if err == nil || !strings.Contains(err.Error(), ".elyro/dev.env") || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("up() error = %v, want actionable env file error", err)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("up() error leaked runtime environment value: %v", err)
	}
	if len(runtime.removes) != 0 || len(runtime.runs) != 0 || len(runtime.starts) != 0 {
		t.Fatalf("preflight failure mutated workspace: removes=%v runs=%v starts=%v", runtime.removes, runtime.runs, runtime.starts)
	}
}

func TestUpReportsProgressAndPreservesPullError(t *testing.T) {
	var progress []string
	_, err := up(t.Context(), &fakeContainerRuntime{pullErr: errors.New("denied: token expired")}, UpRequest{
		ProjectDir:        t.TempDir(),
		SSHConfigPath:     "/tmp/ssh-config",
		Toolchain:         "go",
		ToolchainExplicit: true,
		Progress:          func(message string) { progress = append(progress, message) },
	})
	if err == nil || !strings.Contains(err.Error(), "denied: token expired") {
		t.Fatalf("up() error = %v", err)
	}
	if got, want := strings.Join(progress, "\n"), "Preparing Workspace\nPulling Go Workspace image"; got != want {
		t.Fatalf("progress = %q, want %q", got, want)
	}
	for _, implementationDetail := range []string{"SSH", "registry", "container", "Checking image"} {
		if strings.Contains(strings.Join(progress, "\n"), implementationDetail) {
			t.Fatalf("progress exposes %q: %#v", implementationDetail, progress)
		}
	}
}

func testProjectContext() workspace.ProjectContext {
	return workspace.ProjectContext{
		ProjectDir:    "/tmp/demo",
		Slug:          "demo",
		MountDir:      "/home/elyro/demo",
		ContainerName: "elyro-workspace-demo",
		HostAlias:     "elyro-demo",
	}
}

func testResolvedEnvironment() workspace.ResolvedEnvironment {
	return workspace.ResolvedEnvironment{
		Name:      "python",
		Toolchain: workspace.ToolchainPython,
		Image:     "elyro/workspace-python:latest-amd64",
		Platform:  "linux/amd64",
	}
}

func testContainer(project workspace.ProjectContext, environment workspace.ResolvedEnvironment, status string) *dockerruntime.Container {
	return &dockerruntime.Container{
		ID:          "container-id",
		Name:        project.ContainerName,
		Image:       environment.Image,
		ImageLabel:  environment.Image,
		Status:      status,
		Hostname:    project.Slug,
		ProjectDir:  project.ProjectDir,
		HostAlias:   project.HostAlias,
		Environment: environment.Name,
		Toolchain:   string(environment.Toolchain),
		Platform:    environment.Platform,
		Privileged:  "false",
	}
}

type fakeContainerRuntime struct {
	imageExists bool
	byProject   *dockerruntime.Container
	byName      *dockerruntime.Container
	inspect     map[string]*dockerruntime.Container
	runs        [][]string
	starts      []string
	removes     []string
	waits       []string
	pullErr     error
	pulls       []string
}

func (runtime *fakeContainerRuntime) ImageExists(context.Context, string) bool {
	return runtime.imageExists
}

func (runtime *fakeContainerRuntime) Pull(_ context.Context, image string, _ io.Writer) error {
	runtime.pulls = append(runtime.pulls, image)
	if runtime.pullErr == nil {
		runtime.imageExists = true
	}
	return runtime.pullErr
}

func (runtime *fakeContainerRuntime) InspectByProject(context.Context, string) (*dockerruntime.Container, error) {
	return runtime.byProject, nil
}

func (runtime *fakeContainerRuntime) InspectByName(context.Context, string) (*dockerruntime.Container, error) {
	return runtime.byName, nil
}

func (runtime *fakeContainerRuntime) Inspect(_ context.Context, name string) (*dockerruntime.Container, error) {
	return runtime.inspect[name], nil
}

func (runtime *fakeContainerRuntime) Run(_ context.Context, args ...string) error {
	runtime.runs = append(runtime.runs, append([]string(nil), args...))
	return nil
}

func (runtime *fakeContainerRuntime) Start(_ context.Context, name string) error {
	runtime.starts = append(runtime.starts, name)
	return nil
}

func (runtime *fakeContainerRuntime) Remove(_ context.Context, name string) error {
	runtime.removes = append(runtime.removes, name)
	return nil
}

func (runtime *fakeContainerRuntime) WaitForSSHD(_ context.Context, name string) error {
	runtime.waits = append(runtime.waits, name)
	return nil
}

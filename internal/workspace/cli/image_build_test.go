package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestImageBuildJSONStreamsLogsToStderr(t *testing.T) {
	project := imageBuildTestProject(t)
	originalRun := runImageBuild
	originalStatus := imageBuildWorkspaceStatus
	t.Cleanup(func() { runImageBuild = originalRun; imageBuildWorkspaceStatus = originalStatus })
	var gotDir string
	var gotArgs []string
	runImageBuild = func(_ context.Context, dir string, _ io.Reader, logs io.Writer, args ...string) error {
		gotDir, gotArgs = dir, append([]string(nil), args...)
		_, _ = io.WriteString(logs, "build log\n")
		return nil
	}
	imageBuildWorkspaceStatus = func(context.Context, string) (bool, error) { return false, nil }
	var stdout, stderr bytes.Buffer
	cmd := newImageBuildCmd()
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--project-dir", project, "--platform", "linux/arm64", "--pull", "--no-cache", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"build", "--platform", "linux/arm64", "--pull", "--no-cache", "--progress", "plain", "--file", ".elyro/Dockerfile", "--tag", "elyro-local/demo:dev", "."}
	if gotDir != project || !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("dir = %q args = %#v, want %q %#v", gotDir, gotArgs, project, wantArgs)
	}
	if !strings.Contains(stderr.String(), "build log") || strings.Contains(stdout.String(), "build log") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var payload imageBuildJSONView
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("JSON stdout: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != 1 || payload.Kind != "image" || payload.Action != ImageActionBuilt || payload.Image.Reference != "elyro-local/demo:dev" || payload.Image.Platform != "linux/arm64" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestImageBuildHumanSuggestsRecreateForExistingWorkspace(t *testing.T) {
	project := imageBuildTestProject(t)
	originalRun := runImageBuild
	originalStatus := imageBuildWorkspaceStatus
	t.Cleanup(func() { runImageBuild = originalRun; imageBuildWorkspaceStatus = originalStatus })
	runImageBuild = func(context.Context, string, io.Reader, io.Writer, ...string) error { return nil }
	imageBuildWorkspaceStatus = func(context.Context, string) (bool, error) { return true, nil }
	var stdout bytes.Buffer
	cmd := newImageBuildCmd()
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--project-dir", project})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Workspace image built", "elyro-local/demo:dev", "elyro up --recreate"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestImageBuildSelectsTheOnlyConfiguredEnvironmentWithoutDefault(t *testing.T) {
	project := imageBuildTestProject(t)
	configPath := filepath.Join(project, "elyro.yaml")
	config := strings.Replace(readTestFile(t, configPath), "default_environment: dev\n", "", 1)
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	originalRun := runImageBuild
	originalStatus := imageBuildWorkspaceStatus
	t.Cleanup(func() { runImageBuild = originalRun; imageBuildWorkspaceStatus = originalStatus })
	runImageBuild = func(context.Context, string, io.Reader, io.Writer, ...string) error { return nil }
	imageBuildWorkspaceStatus = func(context.Context, string) (bool, error) { return false, nil }
	var stdout bytes.Buffer
	cmd := newImageBuildCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--project-dir", project, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var payload imageBuildJSONView
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Image.Environment != "dev" {
		t.Fatalf("environment = %q", payload.Image.Environment)
	}
}

func TestImageBuildFailureDoesNotInspectOrChangeWorkspace(t *testing.T) {
	project := imageBuildTestProject(t)
	originalRun := runImageBuild
	originalStatus := imageBuildWorkspaceStatus
	t.Cleanup(func() { runImageBuild = originalRun; imageBuildWorkspaceStatus = originalStatus })
	runImageBuild = func(context.Context, string, io.Reader, io.Writer, ...string) error {
		return errors.New("Dockerfile command failed")
	}
	statusCalled := false
	imageBuildWorkspaceStatus = func(context.Context, string) (bool, error) {
		statusCalled = true
		return true, nil
	}
	var stdout bytes.Buffer
	cmd := newImageBuildCmd()
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--project-dir", project, "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Dockerfile command failed") {
		t.Fatalf("error = %v", err)
	}
	if !statusCalled {
		t.Fatal("image build did not complete its read-only Workspace preflight")
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed JSON build wrote stdout: %q", stdout.String())
	}
}

func TestImageBuildDoesNotReadRuntimeEnvironmentFiles(t *testing.T) {
	project := imageBuildTestProject(t)
	configPath := filepath.Join(project, "elyro.yaml")
	config := readTestFile(t, configPath) + "    docker:\n      env_files:\n        - .elyro/missing-at-runtime.env\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	originalRun := runImageBuild
	originalStatus := imageBuildWorkspaceStatus
	t.Cleanup(func() { runImageBuild = originalRun; imageBuildWorkspaceStatus = originalStatus })
	called := false
	runImageBuild = func(context.Context, string, io.Reader, io.Writer, ...string) error {
		called = true
		return nil
	}
	imageBuildWorkspaceStatus = func(context.Context, string) (bool, error) { return false, nil }
	cmd := newImageBuildCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--project-dir", project, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("image build read a runtime-only env file: %v", err)
	}
	if !called {
		t.Fatal("image build runner was not called")
	}
}

func imageBuildTestProject(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".elyro"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".elyro", "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    image: elyro-local/demo:dev\n    build:\n      context: .\n      dockerfile: .elyro/Dockerfile\n"
	if err := os.WriteFile(filepath.Join(project, "elyro.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

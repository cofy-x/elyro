package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestDoctorJSONSchemaTwo(t *testing.T) {
	var output bytes.Buffer
	report := doctorJSONView{
		SchemaVersion: 2,
		Kind:          "doctor",
		Healthy:       true,
		Project: &doctorProjectView{
			Root: "/tmp/demo", Source: workspace.ProjectRootSourceGit, WorkspaceStatus: "absent",
			ImageBuild: &doctorImageBuildView{Context: ".", Dockerfile: ".elyro/Dockerfile"},
			RuntimeEnvironment: &doctorRuntimeEnvironmentView{
				Variables: []string{"APP_ENV", "CGO_ENABLED"},
				EnvFiles:  []string{".elyro/dev.env", ".elyro/local.env"},
			},
		},
		Checks: []doctorCheck{{
			Scope: "system", Name: "docker_cli", Status: doctorStatusOK, Message: "Docker CLI is available",
		}},
	}
	if err := writeDoctorJSON(&output, report); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["schema_version"] != float64(2) || got["kind"] != "doctor" || got["healthy"] != true {
		t.Fatalf("doctor JSON = %#v", got)
	}
	project := got["project"].(map[string]any)
	imageBuild := project["image_build"].(map[string]any)
	if imageBuild["context"] != "." || imageBuild["dockerfile"] != ".elyro/Dockerfile" {
		t.Fatalf("doctor image_build = %#v", imageBuild)
	}
	runtimeEnvironment := project["runtime_environment"].(map[string]any)
	variables := runtimeEnvironment["variables"].([]any)
	if len(variables) != 2 || variables[0] != "APP_ENV" || variables[1] != "CGO_ENABLED" {
		t.Fatalf("doctor runtime_environment variables = %#v", variables)
	}
	envFiles := runtimeEnvironment["env_files"].([]any)
	if len(envFiles) != 2 || envFiles[0] != ".elyro/dev.env" || envFiles[1] != ".elyro/local.env" {
		t.Fatalf("doctor runtime_environment env_files = %#v", envFiles)
	}
	if strings.Contains(output.String(), "development") || strings.Contains(output.String(), "sha256:") {
		t.Fatalf("doctor JSON leaked runtime environment value or digest: %s", output.String())
	}
	checks := got["checks"].([]any)
	check := checks[0].(map[string]any)
	if _, ok := check["required"]; ok {
		t.Fatalf("doctor check retained removed required field: %#v", check)
	}
	if check["scope"] != "system" || check["status"] != "ok" || check["message"] == "" {
		t.Fatalf("doctor check = %#v", check)
	}
}

func TestDoctorReportHealthFollowsFailures(t *testing.T) {
	report := doctorJSONView{SchemaVersion: 2, Kind: "doctor", Healthy: true}
	report.add(doctorCheck{Scope: "editor", Name: "supported_editor", Status: doctorStatusWarn, Message: "optional"})
	if !report.Healthy {
		t.Fatal("warning made report unhealthy")
	}
	report.add(doctorCheck{Scope: "project", Name: "workspace_configuration", Status: doctorStatusFail, Message: "invalid"})
	if report.Healthy {
		t.Fatal("failure left report healthy")
	}
}

func TestProjectConfigurationExists(t *testing.T) {
	projectDir := t.TempDir()
	if projectConfigurationExists(projectDir) {
		t.Fatal("unconfigured project reported configuration")
	}
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !projectConfigurationExists(projectDir) {
		t.Fatal("configured project did not report configuration")
	}
}

func TestProjectConfigurationExistsForBrokenSymlink(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.Symlink(filepath.Join(projectDir, "missing.yaml"), filepath.Join(projectDir, "elyro.yaml")); err != nil {
		t.Fatal(err)
	}
	if !projectConfigurationExists(projectDir) {
		t.Fatal("broken configuration symlink did not report configuration")
	}
}

func TestPrintDoctorReportGroupsScopes(t *testing.T) {
	var output bytes.Buffer
	report := doctorJSONView{Healthy: true, Checks: []doctorCheck{
		{Scope: "system", Name: "docker_cli", Status: doctorStatusOK, Message: "available"},
		{Scope: "project", Name: "project_root", Status: doctorStatusWarn, Message: "fallback"},
	}}
	if err := printDoctorReport(&output, report); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{"System", "✓ docker cli: available", "Project", "! project root: fallback"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor output %q does not contain %q", got, want)
		}
	}
}

func TestWorkspaceSpecificationMatchesResolvedEnvironment(t *testing.T) {
	project := workspace.ProjectContext{ProjectDir: "/tmp/demo", Slug: "demo", MountDir: "/home/elyro/demo"}
	environment := workspace.ResolvedEnvironment{Name: "go", Toolchain: workspace.ToolchainGo, Image: "example.invalid/go:v1", Platform: "linux/arm64"}
	info := &dockerruntime.Container{
		ProjectDir: project.ProjectDir, Hostname: project.Slug, Environment: environment.Name,
		Toolchain: string(environment.Toolchain), Image: environment.Image, ImageLabel: environment.Image,
		Platform: environment.Platform, Privileged: "false",
	}
	if !local.ContainerSpecificationMatches(info, project, environment, "", "", "false") {
		t.Fatal("matching workspace specification was rejected")
	}
	info.ImageLabel = "example.invalid/other:v1"
	if local.ContainerSpecificationMatches(info, project, environment, "", "", "false") {
		t.Fatal("mismatched image was accepted")
	}
}

func TestWorkspaceSpecificationMatchesRuntimeEnvironmentDigest(t *testing.T) {
	project := workspace.ProjectContext{ProjectDir: "/tmp/demo", Slug: "demo", MountDir: "/home/elyro/demo"}
	environment := workspace.ResolvedEnvironment{
		Name: "go", Toolchain: workspace.ToolchainGo, Image: "example.invalid/go:v1", Platform: "linux/arm64",
		Docker: workspace.DockerOptions{RuntimeEnvironment: workspace.RuntimeEnvironment{Digest: "sha256:current"}},
	}
	info := &dockerruntime.Container{
		ProjectDir: project.ProjectDir, Hostname: project.Slug, Environment: environment.Name,
		Toolchain: string(environment.Toolchain), Image: environment.Image, ImageLabel: environment.Image,
		Platform: environment.Platform, Privileged: "false", RuntimeEnvironmentDigest: "sha256:current",
	}
	if !local.ContainerSpecificationMatches(info, project, environment, "", "", "false") {
		t.Fatal("matching runtime environment digest was rejected")
	}
	info.RuntimeEnvironmentDigest = "sha256:old"
	if local.ContainerSpecificationMatches(info, project, environment, "", "", "false") {
		t.Fatal("mismatched runtime environment digest was accepted")
	}
}

func TestDoctorRuntimeEnvironmentRedactsValues(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectDir, ".elyro"), 0o755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "doctor-runtime-secret-sentinel"
	if err := os.WriteFile(filepath.Join(projectDir, ".elyro", "dev.env"), []byte("FILE_VALUE="+sentinel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    docker:\n      environment:\n        INLINE_VALUE: \"" + sentinel + "\"\n      env_files:\n        - .elyro/dev.env\n"
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	report := doctorJSONView{SchemaVersion: 2, Kind: "doctor", Healthy: true}
	addProjectDoctorChecks(&report, projectDir, true, workspace.Store{}, errors.New("registry unavailable"), false, false)
	if report.Project == nil || report.Project.RuntimeEnvironment == nil {
		t.Fatalf("doctor project = %#v", report.Project)
	}
	if !slices.Equal(report.Project.RuntimeEnvironment.Variables, []string{"FILE_VALUE", "INLINE_VALUE"}) {
		t.Fatalf("runtime variables = %#v", report.Project.RuntimeEnvironment.Variables)
	}
	var output bytes.Buffer
	if err := writeDoctorJSON(&output, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), sentinel) || strings.Contains(output.String(), "sha256:") {
		t.Fatalf("doctor leaked runtime environment value or digest: %s", output.String())
	}
}

func TestDoctorClassifiesInvalidRuntimeEnvironment(t *testing.T) {
	projectDir := t.TempDir()
	config := "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    docker:\n      environment:\n        INVALID: true\n"
	if err := os.WriteFile(filepath.Join(projectDir, "elyro.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	report := doctorJSONView{SchemaVersion: 2, Kind: "doctor", Healthy: true}
	addProjectDoctorChecks(&report, projectDir, true, workspace.Store{}, errors.New("registry unavailable"), false, false)
	for _, check := range report.Checks {
		if check.Name == "runtime_environment" && check.Status == doctorStatusFail {
			return
		}
	}
	t.Fatalf("doctor checks = %#v, want runtime_environment failure", report.Checks)
}

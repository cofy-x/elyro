package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
		Project:       &doctorProjectView{Root: "/tmp/demo", Source: workspace.ProjectRootSourceGit, WorkspaceStatus: "absent"},
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

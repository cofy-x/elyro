package local

import (
	"slices"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func TestDockerRunArgsIncludesRuntimeContract(t *testing.T) {
	t.Parallel()

	project := workspace.ProjectContext{
		ProjectDir:    "/tmp/demo",
		Slug:          "demo",
		MountDir:      "/home/elyro/demo",
		ContainerName: "elyro-workspace-demo",
		HostAlias:     "elyro-demo",
	}
	environment := workspace.ResolvedEnvironment{
		Name:      "python",
		Toolchain: workspace.ToolchainPython,
		Image:     "elyro/workspace-python:latest-amd64",
		Platform:  "linux/amd64",
		Docker: workspace.DockerOptions{
			Privileged: true,
			Mounts: []workspace.DockerMount{
				{Source: "/tmp/cache", Target: "/cache"},
			},
			RuntimeEnvironment: workspace.RuntimeEnvironment{
				Inline: map[string]string{"Z_LAST": "two", "A_FIRST": "one"},
				EnvFiles: []workspace.RuntimeEnvironmentFile{
					{PhysicalPath: "/tmp/demo/.elyro/dev.env"},
					{PhysicalPath: "/tmp/demo/.elyro/local.env"},
				},
				Digest: "sha256:runtime",
			},
		},
	}
	publishes := []workspace.PortPublish{{HostPort: 18080, ContainerPort: 8000}}
	normalizedPublishes := workspace.NormalizePublishSpecs(publishes)
	normalizedMounts := workspace.NormalizeDockerMounts(environment.Docker.Mounts)

	args := dockerRunArgs(project, environment, publishes, normalizedPublishes, normalizedMounts, "true", "22022")

	wantItems := []string{
		"-d",
		"--name", "elyro-workspace-demo",
		"--hostname", "demo",
		"--privileged",
		"--platform", "linux/amd64",
		"-p", "127.0.0.1:18080:8000",
		"-p", "127.0.0.1:22022:22",
		"-v", "/tmp/demo:/home/elyro/demo",
		"-v", "/tmp/cache:/cache",
		"--env-file", "/tmp/demo/.elyro/dev.env",
		"--env-file", "/tmp/demo/.elyro/local.env",
		"--env", "A_FIRST=one",
		"--env", "Z_LAST=two",
		"-w", "/home/elyro/demo",
		"--label", dockerruntime.LabelManaged,
		"--label", dockerruntime.LabelToolchainKey + "=python",
		"--label", dockerruntime.LabelProjectKey + "=/tmp/demo",
		"--label", dockerruntime.LabelPublishKey + "=18080:8000",
		"--label", dockerruntime.LabelPrivileged + "=true",
		"--label", dockerruntime.LabelMountsKey + "=/tmp/cache:/cache",
		"--label", dockerruntime.LabelRuntimeEnvironmentDigest + "=sha256:runtime",
		"elyro/workspace-python:latest-amd64",
	}
	for _, item := range wantItems {
		if !slices.Contains(args, item) {
			t.Fatalf("dockerRunArgs missing %q in %#v", item, args)
		}
	}
	assertOrderedArguments(t, args,
		"--env-file", "/tmp/demo/.elyro/dev.env",
		"--env-file", "/tmp/demo/.elyro/local.env",
		"--env", "A_FIRST=one",
		"--env", "Z_LAST=two",
	)
}

func assertOrderedArguments(t *testing.T, args []string, want ...string) {
	t.Helper()
	position := 0
	for _, argument := range args {
		if position < len(want) && argument == want[position] {
			position++
		}
	}
	if position != len(want) {
		t.Fatalf("arguments %#v do not contain ordered sequence %#v", args, want)
	}
}

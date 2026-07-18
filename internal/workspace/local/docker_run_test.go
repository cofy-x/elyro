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
		"-w", "/home/elyro/demo",
		"--label", dockerruntime.LabelManaged,
		"--label", dockerruntime.LabelToolchainKey + "=python",
		"--label", dockerruntime.LabelProjectKey + "=/tmp/demo",
		"--label", dockerruntime.LabelPublishKey + "=18080:8000",
		"--label", dockerruntime.LabelPrivileged + "=true",
		"--label", dockerruntime.LabelMountsKey + "=/tmp/cache:/cache",
		"elyro/workspace-python:latest-amd64",
	}
	for _, item := range wantItems {
		if !slices.Contains(args, item) {
			t.Fatalf("dockerRunArgs missing %q in %#v", item, args)
		}
	}
}

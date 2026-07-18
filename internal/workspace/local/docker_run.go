package local

import (
	"fmt"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func dockerRunArgs(projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, publishes []workspace.PortPublish, normalizedPublishes, normalizedMounts, privilegedLabel, sshPort string) []string {
	portBinding := "127.0.0.1::22"
	if sshPort != "" {
		portBinding = fmt.Sprintf("127.0.0.1:%s:22", sshPort)
	}

	args := []string{"-d", "--name", projectCtx.ContainerName, "--hostname", projectCtx.Slug}
	if environment.Docker.Privileged {
		args = append(args, "--privileged")
	}
	args = append(args, "--platform", environment.Platform)
	args = append(args, workspace.DockerPublishArgs(publishes)...)
	args = append(args,
		"-p", portBinding,
		"-v", fmt.Sprintf("%s:%s", projectCtx.ProjectDir, projectCtx.MountDir),
	)
	args = append(args, workspace.DockerMountArgs(environment.Docker.Mounts)...)
	args = append(args,
		"-w", projectCtx.MountDir,
		"--label", dockerruntime.LabelManaged,
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelToolchainKey, environment.Toolchain),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelEnvironmentKey, environment.Name),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelImageKey, environment.Image),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelPlatformKey, environment.Platform),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelProjectKey, projectCtx.ProjectDir),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelAliasKey, projectCtx.HostAlias),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelPublishKey, normalizedPublishes),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelPrivileged, privilegedLabel),
		"--label", fmt.Sprintf("%s=%s", dockerruntime.LabelMountsKey, normalizedMounts),
		environment.Image,
	)
	return args
}

package local

import (
	"context"
	"fmt"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func ensureContainer(ctx context.Context, runtime containerRuntime, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, publishes []workspace.PortPublish, normalizedPublishes, normalizedMounts, privilegedLabel, sshPort string) (*dockerruntime.Container, string, error) {
	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return nil, "", err
	}

	if info != nil {
		currentEnvironment := displayEnvironment(info.Environment, info.Toolchain)
		currentImage := info.ImageLabel
		if currentImage == "" {
			currentImage = info.Image
		}
		currentPlatform := displayPlatform(info.Platform)
		if currentEnvironment != environment.Name || currentImage != environment.Image || currentPlatform != environment.Platform || info.Hostname != projectCtx.Slug || info.Published != normalizedPublishes || info.Privileged != privilegedLabel || info.Mounts != normalizedMounts {
			if err := runtime.Remove(ctx, info.Name); err != nil {
				return nil, "", err
			}
			info = nil
		} else if info.Status != "running" {
			if err := runtime.Start(ctx, info.Name); err != nil {
				return nil, "", err
			}
			info, err = runtime.Inspect(ctx, info.Name)
			if err != nil {
				return nil, "", err
			}
			return info, "started", nil
		}
	}

	if info != nil {
		return info, "reused", nil
	}

	occupied, err := runtime.InspectByName(ctx, projectCtx.ContainerName)
	if err != nil {
		return nil, "", err
	}
	if occupied != nil && occupied.ProjectDir != "" && occupied.ProjectDir != projectCtx.ProjectDir {
		return nil, "", fmt.Errorf("container name %s is already in use by project %s", projectCtx.ContainerName, occupied.ProjectDir)
	}
	if occupied != nil && occupied.ProjectDir == "" {
		return nil, "", fmt.Errorf("container name %s is already in use by a non-workspace container", projectCtx.ContainerName)
	}

	if err := runtime.Run(ctx, dockerRunArgs(projectCtx, environment, publishes, normalizedPublishes, normalizedMounts, privilegedLabel, sshPort)...); err != nil {
		return nil, "", err
	}
	info, err = runtime.Inspect(ctx, projectCtx.ContainerName)
	if err != nil {
		_ = runtime.Remove(ctx, projectCtx.ContainerName)
		return nil, "", err
	}
	return info, "created", nil
}

func displayEnvironment(environment, toolchain string) string {
	if environment != "" {
		return environment
	}
	if toolchain != "" {
		return toolchain
	}
	return "unknown"
}

func displayPlatform(platform string) string {
	if platform == "" {
		return workspace.DefaultPlatform
	}
	return platform
}

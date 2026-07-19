package local

import (
	"context"
	"fmt"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func ensureContainer(ctx context.Context, runtime containerRuntime, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, publishes []workspace.PortPublish, normalizedPublishes, normalizedMounts, privilegedLabel, sshPort string, recreate bool) (*dockerruntime.Container, WorkspaceAction, error) {
	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return nil, "", err
	}

	recreating := recreate && info != nil
	if recreating {
		if err := runtime.Remove(ctx, info.Name); err != nil {
			return nil, "", err
		}
		info = nil
	}

	if info != nil {
		if !ContainerSpecificationMatches(info, projectCtx, environment, normalizedPublishes, normalizedMounts, privilegedLabel) {
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
			return info, WorkspaceActionStarted, nil
		}
	}

	if info != nil {
		return info, WorkspaceActionReused, nil
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
	if recreating {
		return info, WorkspaceActionRecreated, nil
	}
	return info, WorkspaceActionCreated, nil
}

func ContainerSpecificationMatches(info *dockerruntime.Container, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, normalizedPublishes, normalizedMounts, privilegedLabel string) bool {
	if info == nil {
		return false
	}
	currentImage := info.ImageLabel
	if currentImage == "" {
		currentImage = info.Image
	}
	return displayEnvironment(info.Environment, info.Toolchain) == environment.Name &&
		currentImage == environment.Image &&
		displayPlatform(info.Platform) == environment.Platform &&
		info.Hostname == projectCtx.Slug &&
		info.Published == normalizedPublishes &&
		info.Privileged == privilegedLabel &&
		info.Mounts == normalizedMounts
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

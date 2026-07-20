package local

import (
	"context"
	"fmt"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

func executeContainerPlan(ctx context.Context, runtime containerRuntime, plan UpPlan, sshPort string) (*dockerruntime.Container, WorkspaceAction, error) {
	info := plan.Container
	switch plan.Action {
	case WorkspacePlanActionRecreate:
		if info == nil {
			return nil, "", fmt.Errorf("invalid recreate plan without an existing workspace")
		}
		if err := runtime.Remove(ctx, info.Name); err != nil {
			return nil, "", err
		}
		info = nil
	case WorkspacePlanActionStart:
		if info == nil {
			return nil, "", fmt.Errorf("invalid start plan without an existing workspace")
		}
		if err := runtime.Start(ctx, info.Name); err != nil {
			return nil, "", err
		}
		started, err := runtime.Inspect(ctx, info.Name)
		return started, WorkspaceActionStarted, err
	case WorkspacePlanActionReuse:
		if info == nil {
			return nil, "", fmt.Errorf("invalid reuse plan without an existing workspace")
		}
		return info, WorkspaceActionReused, nil
	case WorkspacePlanActionCreate:
		if info != nil {
			return nil, "", fmt.Errorf("invalid create plan with an existing workspace")
		}
	default:
		return nil, "", fmt.Errorf("unsupported workspace plan action %q", plan.Action)
	}
	if err := runtime.Run(ctx, dockerRunArgs(plan.Project, plan.Environment, plan.Publishes, plan.Published, plan.Mounts, plan.PrivilegedLabel, sshPort)...); err != nil {
		return nil, "", err
	}
	created, err := runtime.Inspect(ctx, plan.Project.ContainerName)
	if err != nil {
		_ = runtime.Remove(ctx, plan.Project.ContainerName)
		return nil, "", err
	}
	if plan.Action == WorkspacePlanActionRecreate {
		return created, WorkspaceActionRecreated, nil
	}
	return created, WorkspaceActionCreated, nil
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

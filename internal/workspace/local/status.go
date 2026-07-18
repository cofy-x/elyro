package local

import (
	"context"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type StatusRequest struct {
	ProjectDir    string
	ContainerName string
	HostAlias     string
}

type StatusResult struct {
	Project   workspace.ProjectContext
	Container *dockerruntime.Container
}

func Status(ctx context.Context, request StatusRequest) (StatusResult, error) {
	return status(ctx, dockerContainerRuntime{}, request)
}

func status(ctx context.Context, runtime containerRuntime, request StatusRequest) (StatusResult, error) {
	projectCtx, err := resolveProject(request.ProjectDir, request.ContainerName, request.HostAlias)
	if err != nil {
		return StatusResult{}, err
	}
	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{Project: projectCtx, Container: info}, nil
}

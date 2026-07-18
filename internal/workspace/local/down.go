package local

import (
	"context"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type DownRequest struct {
	ProjectDir    string
	SSHConfigPath string
	ContainerName string
	HostAlias     string
}

type DownResult struct {
	Project       workspace.ProjectContext
	Container     *dockerruntime.Container
	ResolvedAlias string
}

func Down(ctx context.Context, request DownRequest) (DownResult, error) {
	return down(ctx, dockerContainerRuntime{}, request)
}

func down(ctx context.Context, runtime containerRuntime, request DownRequest) (DownResult, error) {
	projectCtx, err := resolveProject(request.ProjectDir, request.ContainerName, request.HostAlias)
	if err != nil {
		return DownResult{}, err
	}
	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return DownResult{}, err
	}

	resolvedAlias := projectCtx.HostAlias
	if info != nil {
		if info.HostAlias != "" {
			resolvedAlias = info.HostAlias
		}
		if err := runtime.Remove(ctx, info.Name); err != nil {
			return DownResult{}, err
		}
	}

	if err := workspace.RemoveManagedSSHHost(request.SSHConfigPath, resolvedAlias); err != nil {
		return DownResult{}, err
	}
	if err := workspace.RemoveKnownSSHHost(workspace.DefaultKnownHostsFile(), resolvedAlias); err != nil {
		return DownResult{}, err
	}
	return DownResult{Project: projectCtx, Container: info, ResolvedAlias: resolvedAlias}, nil
}

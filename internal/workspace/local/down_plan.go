package local

import (
	"context"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type DownPlanRequest struct {
	DownRequest
	ManagedSSHPresent bool
	KnownHostPresent  bool
	RegistryPresent   bool
}

type DownPlan struct {
	Project   workspace.ProjectContext
	Container *dockerruntime.Container
	Action    WorkspacePlanAction
	Removes   []WorkspaceManagedResource
	Preserves []WorkspacePreservedResource
}

func PlanDown(ctx context.Context, request DownPlanRequest) (DownPlan, error) {
	return planDown(ctx, dockerContainerRuntime{}, request)
}

func planDown(ctx context.Context, runtime containerRuntime, request DownPlanRequest) (DownPlan, error) {
	projectCtx, err := resolveProject(request.ProjectDir, request.ContainerName, request.HostAlias)
	if err != nil {
		return DownPlan{}, err
	}
	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return DownPlan{}, err
	}
	return CompleteDownPlan(DownPlan{Project: projectCtx, Container: info}, request.ManagedSSHPresent, request.KnownHostPresent, request.RegistryPresent), nil
}

func CompleteDownPlan(plan DownPlan, managedSSHPresent, knownHostPresent, registryPresent bool) DownPlan {
	removes := make([]WorkspaceManagedResource, 0, 4)
	if plan.Container != nil {
		removes = append(removes, WorkspaceManagedResourceContainerWritableLayer)
	}
	if managedSSHPresent {
		removes = append(removes, WorkspaceManagedResourceManagedSSH)
	}
	if knownHostPresent {
		removes = append(removes, WorkspaceManagedResourceKnownHosts)
	}
	if registryPresent {
		removes = append(removes, WorkspaceManagedResourceRegistryRecord)
	}
	action := WorkspacePlanActionNone
	if plan.Container != nil {
		action = WorkspacePlanActionRemove
	} else if len(removes) > 0 {
		action = WorkspacePlanActionCleanup
	}
	plan.Action = action
	plan.Removes = removes
	plan.Preserves = []WorkspacePreservedResource{
		WorkspacePreservedResourceProjectFiles,
		WorkspacePreservedResourceMountedHostData,
		WorkspacePreservedResourceLocalImages,
	}
	return plan
}

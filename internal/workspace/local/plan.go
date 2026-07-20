package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type UpPlan struct {
	Project         workspace.ProjectContext
	Environment     workspace.ResolvedEnvironment
	Container       *dockerruntime.Container
	Action          WorkspacePlanAction
	Reasons         []WorkspaceChangeReason
	ImageStatus     WorkspaceImageStatus
	Publishes       []workspace.PortPublish
	Published       string
	Mounts          string
	PrivilegedLabel string
}

func PlanUp(ctx context.Context, request UpRequest) (UpPlan, error) {
	return planUp(ctx, dockerContainerRuntime{}, request)
}

func planUp(ctx context.Context, runtime containerRuntime, request UpRequest) (UpPlan, error) {
	projectCtx, err := resolveProject(request.ProjectDir, request.ContainerName, request.HostAlias)
	if err != nil {
		return UpPlan{}, err
	}
	resolvedEnvironment, err := workspace.ResolveEnvironment(projectCtx.ProjectDir, projectCtx.MountDir, workspace.EnvironmentSelection{
		Environment:         request.Environment,
		Toolchain:           request.Toolchain,
		Platform:            request.Platform,
		EnvironmentExplicit: request.EnvironmentExplicit,
		ToolchainExplicit:   request.ToolchainExplicit,
		PlatformExplicit:    request.PlatformExplicit,
	})
	if err != nil {
		return UpPlan{}, err
	}
	unsafeReasons := workspace.UnsafeEnvironmentReasons(projectCtx.ProjectDir, resolvedEnvironment.Docker)
	if len(unsafeReasons) > 0 && !request.AllowUnsafeEnvironment {
		return UpPlan{}, fmt.Errorf("unsafe workspace environment requires --allow-unsafe-environment: %s", strings.Join(unsafeReasons, "; "))
	}
	commandPublishes, err := workspace.ParsePublishSpecs(request.PublishSpecs)
	if err != nil {
		return UpPlan{}, err
	}
	publishes, err := workspace.MergePortPublishes(resolvedEnvironment.Docker.Publishes, commandPublishes)
	if err != nil {
		return UpPlan{}, fmt.Errorf("merge environment and command port publishes: %w", err)
	}
	normalizedPublishes := workspace.NormalizePublishSpecs(publishes)
	normalizedMounts := workspace.NormalizeDockerMounts(resolvedEnvironment.Docker.Mounts)
	privilegedLabel := fmt.Sprintf("%t", resolvedEnvironment.Docker.Privileged)
	if err := workspace.ValidateManagedSSHHost(request.SSHConfigPath, projectCtx.HostAlias); err != nil {
		return UpPlan{}, err
	}

	info, err := runtime.InspectByProject(ctx, projectCtx.ProjectDir)
	if err != nil {
		return UpPlan{}, err
	}
	if info == nil {
		occupied, inspectErr := runtime.InspectByName(ctx, projectCtx.ContainerName)
		if inspectErr != nil {
			return UpPlan{}, inspectErr
		}
		if occupied != nil {
			if occupied.ProjectDir != "" {
				return UpPlan{}, fmt.Errorf("container name %s is already in use by project %s", projectCtx.ContainerName, occupied.ProjectDir)
			}
			return UpPlan{}, fmt.Errorf("container name %s is already in use by a non-workspace container", projectCtx.ContainerName)
		}
	}

	imageStatus := WorkspaceImageStatusAvailable
	if !runtime.ImageExists(ctx, resolvedEnvironment.Image) {
		if resolvedEnvironment.Toolchain != "" && !resolvedEnvironment.CustomImage {
			imageStatus = WorkspaceImageStatusPullRequired
		} else if resolvedEnvironment.ImageBuild != nil {
			return UpPlan{}, fmt.Errorf("project Workspace image is missing: %s; run `elyro image build` before `elyro up`", resolvedEnvironment.Image)
		} else {
			return UpPlan{}, fmt.Errorf("missing image %s for %s; build or pull it first", resolvedEnvironment.Image, resolvedEnvironment.Platform)
		}
	}

	action, reasons := planContainerAction(info, projectCtx, resolvedEnvironment, normalizedPublishes, normalizedMounts, privilegedLabel, request.Recreate)
	return UpPlan{
		Project:         projectCtx,
		Environment:     resolvedEnvironment,
		Container:       info,
		Action:          action,
		Reasons:         reasons,
		ImageStatus:     imageStatus,
		Publishes:       publishes,
		Published:       normalizedPublishes,
		Mounts:          normalizedMounts,
		PrivilegedLabel: privilegedLabel,
	}, nil
}

func planContainerAction(info *dockerruntime.Container, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, normalizedPublishes, normalizedMounts, privilegedLabel string, recreate bool) (WorkspacePlanAction, []WorkspaceChangeReason) {
	if info == nil {
		return WorkspacePlanActionCreate, []WorkspaceChangeReason{WorkspaceChangeReasonAbsent}
	}
	if recreate {
		return WorkspacePlanActionRecreate, []WorkspaceChangeReason{WorkspaceChangeReasonExplicitRecreate}
	}
	reasons := CompareContainerSpecification(info, projectCtx, environment, normalizedPublishes, normalizedMounts, privilegedLabel)
	if len(reasons) > 0 {
		return WorkspacePlanActionRecreate, reasons
	}
	if info.Status != "running" {
		return WorkspacePlanActionStart, []WorkspaceChangeReason{WorkspaceChangeReasonStopped}
	}
	return WorkspacePlanActionReuse, []WorkspaceChangeReason{}
}

func CompareContainerSpecification(info *dockerruntime.Container, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, normalizedPublishes, normalizedMounts, privilegedLabel string) []WorkspaceChangeReason {
	if info == nil {
		return []WorkspaceChangeReason{WorkspaceChangeReasonAbsent}
	}
	reasons := make([]WorkspaceChangeReason, 0, 9)
	if displayEnvironment(info.Environment, info.Toolchain) != environment.Name {
		reasons = append(reasons, WorkspaceChangeReasonEnvironment)
	}
	currentImage := info.ImageLabel
	if currentImage == "" {
		currentImage = info.Image
	}
	if currentImage != environment.Image {
		reasons = append(reasons, WorkspaceChangeReasonImage)
	}
	if displayPlatform(info.Platform) != environment.Platform {
		reasons = append(reasons, WorkspaceChangeReasonPlatform)
	}
	if info.Hostname != projectCtx.Slug {
		reasons = append(reasons, WorkspaceChangeReasonHostname)
	}
	if info.Published != normalizedPublishes {
		reasons = append(reasons, WorkspaceChangeReasonPublishedPorts)
	}
	if info.Mounts != normalizedMounts {
		reasons = append(reasons, WorkspaceChangeReasonMounts)
	}
	if info.Privileged != privilegedLabel {
		reasons = append(reasons, WorkspaceChangeReasonPrivileged)
	}
	if info.RuntimeEnvironmentDigest != environment.Docker.RuntimeEnvironment.Digest {
		reasons = append(reasons, WorkspaceChangeReasonRuntimeEnvironment)
	}
	return reasons
}

func ContainerSpecificationMatches(info *dockerruntime.Container, projectCtx workspace.ProjectContext, environment workspace.ResolvedEnvironment, normalizedPublishes, normalizedMounts, privilegedLabel string) bool {
	return len(CompareContainerSpecification(info, projectCtx, environment, normalizedPublishes, normalizedMounts, privilegedLabel)) == 0
}

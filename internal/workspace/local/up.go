package local

import (
	"context"
	"fmt"
	"io"

	elyroversion "github.com/cofy-x/elyro/internal/version"
	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/access"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type UpRequest struct {
	ProjectDir             string
	SSHConfigPath          string
	IdentityFile           string
	AllowUnsafeEnvironment bool
	ContainerName          string
	HostAlias              string
	Toolchain              string
	Environment            string
	Platform               string
	ToolchainExplicit      bool
	EnvironmentExplicit    bool
	PlatformExplicit       bool
	SSHPort                string
	PublishSpecs           []string
	Recreate               bool
	PullOutput             io.Writer
	Progress               func(string)
}

type UpResult struct {
	Project         workspace.ProjectContext
	Environment     workspace.ResolvedEnvironment
	Container       dockerruntime.Container
	SSHConfigPath   string
	IdentityFile    string
	Action          WorkspaceAction
	Reasons         []WorkspaceChangeReason
	PrivilegedLabel string
	Published       string
	Mounts          string
}

func Up(ctx context.Context, request UpRequest) (result UpResult, err error) {
	return up(ctx, dockerContainerRuntime{}, request)
}

func up(ctx context.Context, runtime containerRuntime, request UpRequest) (result UpResult, err error) {
	reportProgress(request, "Preparing Workspace")
	plan, err := planUp(ctx, runtime, request)
	if err != nil {
		return UpResult{}, err
	}
	if plan.ImageStatus == WorkspaceImageStatusPullRequired {
		puller, ok := runtime.(imagePuller)
		if !ok {
			return UpResult{}, fmt.Errorf("missing image %s for %s; build or pull it first", plan.Environment.Image, plan.Environment.Platform)
		}
		reportProgress(request, "Pulling "+workspaceImageDisplayName(plan.Environment)+" Workspace image")
		if pullErr := puller.Pull(ctx, plan.Environment.Image, request.PullOutput); pullErr != nil {
			return UpResult{}, imagePullError(plan.Environment, pullErr)
		}
		if !runtime.ImageExists(ctx, plan.Environment.Image) {
			return UpResult{}, fmt.Errorf("missing image %s for %s after pull", plan.Environment.Image, plan.Environment.Platform)
		}
	}
	resolvedIdentityFile, publicKey, err := access.EnsureSSHIdentity(request.IdentityFile)
	if err != nil {
		return UpResult{}, err
	}

	reportProgress(request, "Starting Workspace")
	info, action, err := executeContainerPlan(ctx, runtime, plan, request.SSHPort)
	if err != nil {
		return UpResult{}, err
	}
	defer func() {
		if err == nil || !action.CreatedContainer() {
			return
		}
		_ = runtime.Remove(ctx, info.Name)
	}()

	if err := runtime.WaitForSSHD(ctx, info.Name); err != nil {
		return UpResult{}, err
	}
	if err := access.InstallContainerSSHAccess(ctx, info.Name, publicKey, plan.Environment.Docker.RuntimeEnvironment.Effective); err != nil {
		return UpResult{}, err
	}
	knownHostsFile := workspace.DefaultKnownHostsFile()
	if err := workspace.PrepareKnownSSHHost(ctx, knownHostsFile, info.HostAlias, info.ID, "127.0.0.1", info.HostPort); err != nil {
		return UpResult{}, err
	}
	if err := workspace.UpsertManagedSSHHost(request.SSHConfigPath, workspace.SSHHostEntry{
		HostAlias:      info.HostAlias,
		HostName:       "127.0.0.1",
		Port:           info.HostPort,
		User:           "elyro",
		IdentityFile:   resolvedIdentityFile,
		KnownHostsFile: knownHostsFile,
	}); err != nil {
		return UpResult{}, err
	}

	if plan.Environment.ProjectConfigured {
		if err := workspace.EnsureVSCodeWorkspace(plan.Project.ProjectDir, plan.Environment); err != nil {
			return UpResult{}, err
		}
	}

	return UpResult{
		Project:         plan.Project,
		Environment:     plan.Environment,
		Container:       *info,
		SSHConfigPath:   request.SSHConfigPath,
		IdentityFile:    resolvedIdentityFile,
		Action:          action,
		Reasons:         plan.Reasons,
		PrivilegedLabel: plan.PrivilegedLabel,
		Published:       plan.Published,
		Mounts:          plan.Mounts,
	}, nil
}

func workspaceImageDisplayName(environment workspace.ResolvedEnvironment) string {
	switch environment.Toolchain {
	case workspace.ToolchainPython:
		return "Python"
	case workspace.ToolchainGo:
		return "Go"
	case workspace.ToolchainNode:
		return "Node.js"
	default:
		return "custom"
	}
}

func reportProgress(request UpRequest, message string) {
	if request.Progress != nil {
		request.Progress(message)
	}
}

func imagePullError(environment workspace.ResolvedEnvironment, pullErr error) error {
	if elyroversion.IsRelease() {
		return fmt.Errorf("pull image %s: %w; check network access, registry credentials, and ELYRO_IMAGE_PREFIX", environment.Image, pullErr)
	}
	if environment.Toolchain != "" {
		return fmt.Errorf("pull image %s: %w; build or pull the image before retrying", environment.Image, pullErr)
	}
	return fmt.Errorf("pull image %s: %w", environment.Image, pullErr)
}

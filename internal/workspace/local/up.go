package local

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	PrivilegedLabel string
	Published       string
	Mounts          string
}

func Up(ctx context.Context, request UpRequest) (result UpResult, err error) {
	return up(ctx, dockerContainerRuntime{}, request)
}

func up(ctx context.Context, runtime containerRuntime, request UpRequest) (result UpResult, err error) {
	reportProgress(request, "Preparing Workspace")
	projectCtx, err := resolveProject(request.ProjectDir, request.ContainerName, request.HostAlias)
	if err != nil {
		return UpResult{}, err
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
		return UpResult{}, err
	}
	unsafeReasons := workspace.UnsafeEnvironmentReasons(projectCtx.ProjectDir, resolvedEnvironment.Docker)
	if len(unsafeReasons) > 0 && !request.AllowUnsafeEnvironment {
		return UpResult{}, fmt.Errorf("unsafe workspace environment requires --allow-unsafe-environment: %s", strings.Join(unsafeReasons, "; "))
	}

	imageAvailable := runtime.ImageExists(ctx, resolvedEnvironment.Image)
	if !imageAvailable {
		if resolvedEnvironment.Toolchain != "" && !resolvedEnvironment.CustomImage {
			if puller, ok := runtime.(imagePuller); ok {
				reportProgress(request, "Pulling "+workspaceImageDisplayName(resolvedEnvironment)+" Workspace image")
				if pullErr := puller.Pull(ctx, resolvedEnvironment.Image, request.PullOutput); pullErr != nil {
					return UpResult{}, imagePullError(resolvedEnvironment, pullErr)
				}
				imageAvailable = runtime.ImageExists(ctx, resolvedEnvironment.Image)
			}
		}
	}
	if !imageAvailable {
		if resolvedEnvironment.ImageBuild != nil {
			return UpResult{}, fmt.Errorf("project Workspace image is missing: %s; run `elyro image build` before `elyro up`", resolvedEnvironment.Image)
		}
		if !resolvedEnvironment.CustomImage && resolvedEnvironment.Name == string(resolvedEnvironment.Toolchain) && resolvedEnvironment.Toolchain != "" {
			return UpResult{}, fmt.Errorf("missing image %s for %s; build or pull it first", resolvedEnvironment.Image, resolvedEnvironment.Platform)
		}
		return UpResult{}, fmt.Errorf("missing image %s for %s; build or pull it first", resolvedEnvironment.Image, resolvedEnvironment.Platform)
	}
	if err := workspace.ValidateManagedSSHHost(request.SSHConfigPath, projectCtx.HostAlias); err != nil {
		return UpResult{}, err
	}
	resolvedIdentityFile, publicKey, err := access.EnsureSSHIdentity(request.IdentityFile)
	if err != nil {
		return UpResult{}, err
	}
	commandPublishes, err := workspace.ParsePublishSpecs(request.PublishSpecs)
	if err != nil {
		return UpResult{}, err
	}
	publishes, err := workspace.MergePortPublishes(resolvedEnvironment.Docker.Publishes, commandPublishes)
	if err != nil {
		return UpResult{}, fmt.Errorf("merge environment and command port publishes: %w", err)
	}
	normalizedPublishes := workspace.NormalizePublishSpecs(publishes)
	normalizedMounts := workspace.NormalizeDockerMounts(resolvedEnvironment.Docker.Mounts)
	privilegedLabel := fmt.Sprintf("%t", resolvedEnvironment.Docker.Privileged)

	reportProgress(request, "Starting Workspace")
	info, action, err := ensureContainer(ctx, runtime, projectCtx, resolvedEnvironment, publishes, normalizedPublishes, normalizedMounts, privilegedLabel, request.SSHPort, request.Recreate)
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
	if err := access.InstallContainerSSHAccess(ctx, info.Name, publicKey); err != nil {
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

	if resolvedEnvironment.ProjectConfigured {
		if err := workspace.EnsureVSCodeWorkspace(projectCtx.ProjectDir, resolvedEnvironment); err != nil {
			return UpResult{}, err
		}
	}

	return UpResult{
		Project:         projectCtx,
		Environment:     resolvedEnvironment,
		Container:       *info,
		SSHConfigPath:   request.SSHConfigPath,
		IdentityFile:    resolvedIdentityFile,
		Action:          action,
		PrivilegedLabel: privilegedLabel,
		Published:       normalizedPublishes,
		Mounts:          normalizedMounts,
	}, nil
}

func workspaceImageDisplayName(environment workspace.ResolvedEnvironment) string {
	switch environment.Toolchain {
	case workspace.ToolchainPython:
		return "Python"
	case workspace.ToolchainGo:
		return "Go"
	case workspace.ToolchainJava:
		return "Java"
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

package local

import (
	"context"
	"io"

	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type containerRuntime interface {
	ImageExists(ctx context.Context, image string) bool
	InspectByProject(ctx context.Context, projectDir string) (*dockerruntime.Container, error)
	InspectByName(ctx context.Context, name string) (*dockerruntime.Container, error)
	Inspect(ctx context.Context, name string) (*dockerruntime.Container, error)
	Run(ctx context.Context, args ...string) error
	Start(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
	WaitForSSHD(ctx context.Context, name string) error
}

type imagePuller interface {
	Pull(context.Context, string, io.Writer) error
}

type dockerContainerRuntime struct{}

func (dockerContainerRuntime) ImageExists(ctx context.Context, image string) bool {
	return dockerruntime.ImageExists(ctx, image)
}

func (dockerContainerRuntime) Pull(ctx context.Context, image string, output io.Writer) error {
	return dockerruntime.Pull(ctx, image, output)
}

func (dockerContainerRuntime) InspectByProject(ctx context.Context, projectDir string) (*dockerruntime.Container, error) {
	return dockerruntime.InspectByProject(ctx, projectDir)
}

func (dockerContainerRuntime) InspectByName(ctx context.Context, name string) (*dockerruntime.Container, error) {
	return dockerruntime.InspectByName(ctx, name)
}

func (dockerContainerRuntime) Inspect(ctx context.Context, name string) (*dockerruntime.Container, error) {
	return dockerruntime.Inspect(ctx, name)
}

func (dockerContainerRuntime) Run(ctx context.Context, args ...string) error {
	return dockerruntime.Run(ctx, args...)
}

func (dockerContainerRuntime) Start(ctx context.Context, name string) error {
	return dockerruntime.Start(ctx, name)
}

func (dockerContainerRuntime) Remove(ctx context.Context, name string) error {
	return dockerruntime.Remove(ctx, name)
}

func (dockerContainerRuntime) WaitForSSHD(ctx context.Context, name string) error {
	return dockerruntime.WaitForSSHD(ctx, name)
}

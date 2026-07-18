package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func ImageExists(ctx context.Context, image string) bool {
	_, err := runOutput(ctx, "docker", "image", "inspect", image)
	return err == nil
}

func Pull(ctx context.Context, image string, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	var stderr bytes.Buffer
	cmd.Stdout = output
	cmd.Stderr = io.MultiWriter(output, &stderr)
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return fmt.Errorf("docker pull %s: %w: %s", image, err, message)
		}
		return fmt.Errorf("docker pull %s: %w", image, err)
	}
	return nil
}

func Run(ctx context.Context, args ...string) error {
	allArgs := append([]string{"run"}, args...)
	_, err := runOutput(ctx, "docker", allArgs...)
	return err
}

func Start(ctx context.Context, containerName string) error {
	_, err := runOutput(ctx, "docker", "start", containerName)
	return err
}

func Remove(ctx context.Context, containerName string) error {
	_, err := runOutput(ctx, "docker", "rm", "-f", containerName)
	return err
}

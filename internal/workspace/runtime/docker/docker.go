package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	LabelManaged                  = "elyro.workspace.managed=true"
	LabelToolchainKey             = "elyro.workspace.toolchain"
	LabelEnvironmentKey           = "elyro.workspace.environment"
	LabelImageKey                 = "elyro.workspace.image"
	LabelPlatformKey              = "elyro.workspace.platform"
	LabelProjectKey               = "elyro.workspace.project_dir"
	LabelAliasKey                 = "elyro.workspace.host_alias"
	LabelPublishKey               = "elyro.workspace.publish"
	LabelPrivileged               = "elyro.workspace.privileged"
	LabelMountsKey                = "elyro.workspace.mounts"
	LabelRuntimeEnvironmentDigest = "elyro.workspace.runtime_environment_digest"
)

type Container struct {
	ID                       string
	Name                     string
	Image                    string
	Status                   string
	Hostname                 string
	Toolchain                string
	Environment              string
	ImageLabel               string
	Platform                 string
	ProjectDir               string
	HostAlias                string
	HostPort                 string
	Published                string
	Privileged               string
	Mounts                   string
	RuntimeEnvironmentDigest string
}

func InspectByProject(ctx context.Context, projectDir string) (*Container, error) {
	idsOutput, err := runOutput(ctx, "docker", "ps", "-aq",
		"--filter", "label="+LabelManaged,
		"--filter", fmt.Sprintf("label=%s=%s", LabelProjectKey, projectDir),
	)
	if err != nil {
		return nil, err
	}
	ids := filterNonEmpty(strings.Split(strings.TrimSpace(idsOutput), "\n"))
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("expected at most one workspace container for %s, found %d", projectDir, len(ids))
	}
	return Inspect(ctx, ids[0])
}

func ListRunning(ctx context.Context) ([]Container, error) {
	idsOutput, err := runOutput(ctx, "docker", "ps", "-q",
		"--filter", "label="+LabelManaged,
	)
	if err != nil {
		return nil, err
	}
	ids := filterNonEmpty(strings.Split(strings.TrimSpace(idsOutput), "\n"))
	containers := make([]Container, 0, len(ids))
	for _, id := range ids {
		info, err := Inspect(ctx, id)
		if err != nil {
			return nil, err
		}
		containers = append(containers, *info)
	}
	return containers, nil
}

func InspectByName(ctx context.Context, name string) (*Container, error) {
	info, err := Inspect(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "No such object") {
			return nil, nil
		}
		return nil, err
	}
	return info, nil
}

func Inspect(ctx context.Context, idOrName string) (*Container, error) {
	format := "{{.Id}}\t{{.Name}}\t{{.Config.Image}}\t{{.State.Status}}\t{{.Config.Hostname}}\t{{index .Config.Labels \"" + LabelToolchainKey + "\"}}\t{{index .Config.Labels \"" + LabelEnvironmentKey + "\"}}\t{{index .Config.Labels \"" + LabelImageKey + "\"}}\t{{index .Config.Labels \"" + LabelPlatformKey + "\"}}\t{{index .Config.Labels \"" + LabelProjectKey + "\"}}\t{{index .Config.Labels \"" + LabelAliasKey + "\"}}\t{{with (index .NetworkSettings.Ports \"22/tcp\")}}{{(index . 0).HostPort}}{{end}}\t{{index .Config.Labels \"" + LabelPublishKey + "\"}}\t{{index .Config.Labels \"" + LabelPrivileged + "\"}}\t{{index .Config.Labels \"" + LabelMountsKey + "\"}}\t{{with (index .Config.Labels \"" + LabelRuntimeEnvironmentDigest + "\")}}{{.}}{{end}}"
	output, err := runOutput(ctx, "docker", "inspect", "--format", format, idOrName)
	if err != nil {
		return nil, err
	}
	return parseInspectOutput(idOrName, output)
}

func WaitForSSHD(ctx context.Context, containerName string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		_, err := runOutput(ctx, "docker", "exec", containerName, "pgrep", "-x", "sshd")
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("sshd did not become ready in container %s", containerName)
}

func parseInspectOutput(idOrName, output string) (*Container, error) {
	fields := strings.Split(strings.TrimRight(output, "\r\n"), "\t")
	if len(fields) < 16 {
		return nil, fmt.Errorf("unexpected docker inspect output for %s", idOrName)
	}
	return &Container{
		ID:                       fields[0],
		Name:                     strings.TrimPrefix(fields[1], "/"),
		Image:                    fields[2],
		Status:                   fields[3],
		Hostname:                 fields[4],
		Toolchain:                fields[5],
		Environment:              fields[6],
		ImageLabel:               fields[7],
		Platform:                 fields[8],
		ProjectDir:               fields[9],
		HostAlias:                fields[10],
		HostPort:                 fields[11],
		Published:                fields[12],
		Privileged:               fields[13],
		Mounts:                   fields[14],
		RuntimeEnvironmentDigest: fields[15],
	}, nil
}

func runOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

func filterNonEmpty(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

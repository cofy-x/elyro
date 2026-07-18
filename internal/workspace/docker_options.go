package workspace

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type DockerOptions struct {
	Privileged bool
	Mounts     []DockerMount
	Publishes  []PortPublish
}

type DockerMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

type configuredDocker struct {
	Privileged bool              `yaml:"privileged"`
	Mounts     []configuredMount `yaml:"mounts"`
	Publish    []string          `yaml:"publish"`
}

type configuredMount struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"read_only"`
}

func resolveDockerOptions(projectDir string, cfg configuredDocker) (DockerOptions, error) {
	publishes, err := ParsePublishSpecs(cfg.Publish)
	if err != nil {
		return DockerOptions{}, fmt.Errorf("docker.publish: %w", err)
	}
	mounts := make([]DockerMount, 0, len(cfg.Mounts))
	for _, rawMount := range cfg.Mounts {
		source := strings.TrimSpace(rawMount.Source)
		target := strings.TrimSpace(rawMount.Target)
		if source == "" {
			return DockerOptions{}, errors.New("docker.mounts[].source must not be empty")
		}
		if target == "" {
			return DockerOptions{}, errors.New("docker.mounts[].target must not be empty")
		}
		if !path.IsAbs(target) {
			return DockerOptions{}, fmt.Errorf("docker.mounts[].target must be an absolute path, got %q", target)
		}

		resolvedSource, err := ExpandPath(source)
		if err != nil {
			return DockerOptions{}, fmt.Errorf("expand docker mount source %q: %w", source, err)
		}
		if !filepath.IsAbs(resolvedSource) {
			resolvedSource = filepath.Join(projectDir, resolvedSource)
		}
		resolvedSource, err = filepath.Abs(resolvedSource)
		if err != nil {
			return DockerOptions{}, fmt.Errorf("resolve docker mount source %q: %w", source, err)
		}

		mounts = append(mounts, DockerMount{
			Source:   resolvedSource,
			Target:   target,
			ReadOnly: rawMount.ReadOnly,
		})
	}

	sort.Slice(mounts, func(i, j int) bool {
		if mounts[i].Source != mounts[j].Source {
			return mounts[i].Source < mounts[j].Source
		}
		if mounts[i].Target != mounts[j].Target {
			return mounts[i].Target < mounts[j].Target
		}
		return !mounts[i].ReadOnly && mounts[j].ReadOnly
	})

	return DockerOptions{
		Privileged: cfg.Privileged,
		Mounts:     mounts,
		Publishes:  publishes,
	}, nil
}

func NormalizeDockerMounts(mounts []DockerMount) string {
	if len(mounts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		spec := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
		if mount.ReadOnly {
			spec += ":ro"
		}
		parts = append(parts, spec)
	}
	return strings.Join(parts, ",")
}

func DockerMountArgs(mounts []DockerMount) []string {
	args := make([]string, 0, len(mounts)*2)
	for _, mount := range mounts {
		spec := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
		if mount.ReadOnly {
			spec += ":ro"
		}
		args = append(args, "-v", spec)
	}
	return args
}

func UnsafeEnvironmentReasons(projectDir string, options DockerOptions) []string {
	reasons := make([]string, 0)
	if options.Privileged {
		reasons = append(reasons, "privileged container access")
	}
	projectRoot, _ := filepath.Abs(projectDir)
	for _, mount := range options.Mounts {
		cleanSource := filepath.Clean(mount.Source)
		if cleanSource == "/var/run/docker.sock" || cleanSource == "/run/docker.sock" {
			reasons = append(reasons, "Docker socket mount "+cleanSource)
			continue
		}
		rel, err := filepath.Rel(projectRoot, cleanSource)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			reasons = append(reasons, "host mount outside project "+cleanSource)
		}
	}
	return reasons
}

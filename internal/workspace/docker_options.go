package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type DockerOptions struct {
	Privileged         bool
	Mounts             []DockerMount
	Publishes          []PortPublish
	RuntimeEnvironment RuntimeEnvironment
}

type DockerMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

type configuredDocker struct {
	Privileged  bool                           `yaml:"privileged"`
	Mounts      []configuredMount              `yaml:"mounts"`
	Publish     []string                       `yaml:"publish"`
	Environment configuredEnvironmentVariables `yaml:"environment"`
	EnvFiles    configuredEnvironmentFiles     `yaml:"env_files"`
}

func (docker *configuredDocker) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("docker must be a mapping")
	}
	seen := make(map[string]struct{}, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if _, exists := seen[key]; exists {
			return fmt.Errorf("docker field %q is duplicated", key)
		}
		seen[key] = struct{}{}
		switch key {
		case "environment":
			if value.Kind != yaml.MappingNode {
				return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.environment must be a string mapping")}
			}
		case "env_files":
			if value.Kind != yaml.SequenceNode {
				return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.env_files must be a string list")}
			}
		}
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	type plainDocker configuredDocker
	var decoded plainDocker
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*docker = configuredDocker(decoded)
	return nil
}

type configuredMount struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"read_only"`
}

func resolveDockerOptions(projectDir string, cfg configuredDocker, skipRuntimeEnvironment bool) (DockerOptions, error) {
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
	var runtimeEnvironment RuntimeEnvironment
	if !skipRuntimeEnvironment {
		runtimeEnvironment, err = ResolveRuntimeEnvironment(projectDir, cfg.Environment.Values, cfg.EnvFiles.Paths)
		if err != nil {
			return DockerOptions{}, &RuntimeEnvironmentError{Err: err}
		}
	}

	return DockerOptions{
		Privileged:         cfg.Privileged,
		Mounts:             mounts,
		Publishes:          publishes,
		RuntimeEnvironment: runtimeEnvironment,
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

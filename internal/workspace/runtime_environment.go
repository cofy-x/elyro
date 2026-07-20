package workspace

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var environmentVariableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// RuntimeEnvironment is a validated container-wide environment contract.
// Values remain in memory only and must not be copied into public views,
// registry records, logs, or container labels.
type RuntimeEnvironment struct {
	Inline        map[string]string
	EnvFiles      []RuntimeEnvironmentFile
	VariableNames []string
	Digest        string
	Configured    bool
	// Effective contains validated final values for container creation and
	// managed SSH setup. It must never enter public views, logs, or registry data.
	Effective map[string]string
}

type RuntimeEnvironmentFile struct {
	RelativePath string
	PhysicalPath string
}

type RuntimeEnvironmentError struct {
	Err error
}

func (e *RuntimeEnvironmentError) Error() string { return e.Err.Error() }
func (e *RuntimeEnvironmentError) Unwrap() error { return e.Err }

func IsRuntimeEnvironmentError(err error) bool {
	var target *RuntimeEnvironmentError
	return errors.As(err, &target)
}

type configuredEnvironmentVariables struct {
	Values map[string]string
}

func (variables *configuredEnvironmentVariables) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode || node.Tag == "!!null" {
		return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.environment must be a string mapping")}
	}
	variables.Values = make(map[string]string, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode, valueNode := node.Content[i], node.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
			return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.environment keys must be strings")}
		}
		key := keyNode.Value
		if _, exists := variables.Values[key]; exists {
			return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.environment variable %q is duplicated", key)}
		}
		if valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!str" {
			return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.environment.%s must be a string", key)}
		}
		variables.Values[key] = valueNode.Value
	}
	return nil
}

type configuredEnvironmentFiles struct {
	Paths []string
}

func (files *configuredEnvironmentFiles) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode || node.Tag == "!!null" {
		return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.env_files must be a string list")}
	}
	files.Paths = make([]string, 0, len(node.Content))
	for i, item := range node.Content {
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			return &RuntimeEnvironmentError{Err: fmt.Errorf("docker.env_files[%d] must be a string", i)}
		}
		files.Paths = append(files.Paths, item.Value)
	}
	return nil
}

func ResolveRuntimeEnvironment(projectDir string, inline map[string]string, configuredFiles []string) (RuntimeEnvironment, error) {
	result := RuntimeEnvironment{
		Inline:     make(map[string]string, len(inline)),
		Configured: inline != nil || configuredFiles != nil,
	}
	if configuredFiles != nil {
		result.EnvFiles = make([]RuntimeEnvironmentFile, 0, len(configuredFiles))
	}
	effective := make(map[string]string)
	seenPaths := make(map[string]struct{}, len(configuredFiles))

	for _, rawPath := range configuredFiles {
		relative, physical, err := resolveBuildPath(projectDir, rawPath, false)
		if err != nil {
			return RuntimeEnvironment{}, fmt.Errorf("docker.env_files path %q: %w", rawPath, err)
		}
		if _, exists := seenPaths[physical]; exists {
			return RuntimeEnvironment{}, fmt.Errorf("docker.env_files path %q is duplicated", relative)
		}
		seenPaths[physical] = struct{}{}
		values, err := parseRuntimeEnvironmentFile(relative, physical)
		if err != nil {
			return RuntimeEnvironment{}, err
		}
		for key, value := range values {
			effective[key] = value
		}
		result.EnvFiles = append(result.EnvFiles, RuntimeEnvironmentFile{RelativePath: relative, PhysicalPath: physical})
	}

	for key, value := range inline {
		if err := validateRuntimeEnvironmentEntry(key, value); err != nil {
			return RuntimeEnvironment{}, fmt.Errorf("docker.environment: %w", err)
		}
		result.Inline[key] = value
		effective[key] = value
	}

	result.VariableNames = sortedEnvironmentVariableNames(effective)
	result.Digest = runtimeEnvironmentDigest(effective, result.VariableNames)
	result.Effective = effective
	return result, nil
}

func parseRuntimeEnvironmentFile(relative, physical string) (map[string]string, error) {
	data, err := os.ReadFile(physical)
	if err != nil {
		return nil, fmt.Errorf("read docker.env_files path %q: %w", relative, err)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("docker.env_files path %q must be UTF-8", relative)
	}
	values := make(map[string]string)
	for index, rawLine := range bytes.Split(data, []byte{'\n'}) {
		line := strings.TrimSuffix(string(rawLine), "\r")
		lineNumber := index + 1
		if strings.ContainsRune(line, '\r') {
			return nil, fmt.Errorf("docker.env_files path %q line %d contains an unsupported carriage return", relative, lineNumber)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		equals := strings.IndexByte(line, '=')
		if equals < 0 {
			return nil, fmt.Errorf("docker.env_files path %q line %d must use KEY=VALUE; bare variables and host inheritance are not supported", relative, lineNumber)
		}
		key, value := line[:equals], line[equals+1:]
		if err := validateRuntimeEnvironmentEntry(key, value); err != nil {
			return nil, fmt.Errorf("docker.env_files path %q line %d: %w", relative, lineNumber, err)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("docker.env_files path %q line %d duplicates variable %q", relative, lineNumber, key)
		}
		values[key] = value
	}
	return values, nil
}

func validateRuntimeEnvironmentEntry(key, value string) error {
	if !environmentVariableName.MatchString(key) {
		return fmt.Errorf("invalid environment variable name %q", key)
	}
	if strings.ContainsRune(value, 0) || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("environment variable %q contains NUL or a newline", key)
	}
	return nil
}

func sortedEnvironmentVariableNames(values map[string]string) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func runtimeEnvironmentDigest(values map[string]string, names []string) string {
	if len(names) == 0 {
		return ""
	}
	hash := sha256.New()
	var size [8]byte
	for _, name := range names {
		value := values[name]
		binary.BigEndian.PutUint64(size[:], uint64(len(name)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write([]byte(name))
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write([]byte(value))
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func DockerRuntimeEnvironmentArgs(environment RuntimeEnvironment) []string {
	args := make([]string, 0, len(environment.EnvFiles)*2+len(environment.Inline)*2)
	for _, file := range environment.EnvFiles {
		args = append(args, "--env-file", file.PhysicalPath)
	}
	names := sortedEnvironmentVariableNames(environment.Inline)
	for _, name := range names {
		args = append(args, "--env", name+"="+environment.Inline[name])
	}
	return args
}

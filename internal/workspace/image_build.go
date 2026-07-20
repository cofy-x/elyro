package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
)

// ImageBuild is a validated, project-owned Docker build definition.
type ImageBuild struct {
	Context    string
	Dockerfile string
}

// ResolveImageBuild validates a configured image build and resolves its paths.
func ResolveImageBuild(projectDir, image, contextPath, dockerfilePath string) (ImageBuild, error) {
	if err := ValidateBuildImageReference(image); err != nil {
		return ImageBuild{}, err
	}
	contextRelative, _, err := resolveBuildPath(projectDir, contextPath, true)
	if err != nil {
		return ImageBuild{}, fmt.Errorf("build context: %w", err)
	}
	dockerfileRelative, _, err := resolveBuildPath(projectDir, dockerfilePath, false)
	if err != nil {
		return ImageBuild{}, fmt.Errorf("build dockerfile: %w", err)
	}
	return ImageBuild{Context: contextRelative, Dockerfile: dockerfileRelative}, nil
}

// ValidateBuildImageReference applies the stricter contract used only for
// Elyro-managed build targets. Prebuilt custom images are intentionally not
// subject to this policy.
func ValidateBuildImageReference(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fmt.Errorf("build image must not be empty")
	}
	named, err := reference.ParseNormalizedNamed(value)
	if err != nil {
		return fmt.Errorf("invalid build image %q: %w", raw, err)
	}
	if _, ok := named.(reference.Digested); ok {
		return fmt.Errorf("build image %q must not use a digest", raw)
	}
	tagged, ok := named.(reference.NamedTagged)
	if !ok {
		return fmt.Errorf("build image %q must include an explicit tag", raw)
	}
	if tagged.Tag() == "latest" {
		return fmt.Errorf("build image %q must not use the latest tag", raw)
	}
	return nil
}

func resolveBuildPath(projectDir, raw string, wantDirectory bool) (string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", fmt.Errorf("path must not be empty")
	}
	if filepath.IsAbs(value) {
		return "", "", fmt.Errorf("path must be project-relative: %s", value)
	}
	clean := filepath.Clean(value)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes the project: %s", value)
	}
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve project root: %w", err)
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve project root: %w", err)
	}
	absolute := filepath.Join(root, clean)
	physical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", "", fmt.Errorf("resolve %s: %w", clean, err)
	}
	contained, err := filepath.Rel(physicalRoot, physical)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path resolves outside the project: %s", clean)
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", "", fmt.Errorf("inspect %s: %w", clean, err)
	}
	if wantDirectory && !info.IsDir() {
		return "", "", fmt.Errorf("path is not a directory: %s", clean)
	}
	if !wantDirectory && (!info.Mode().IsRegular()) {
		return "", "", fmt.Errorf("path is not a regular file: %s", clean)
	}
	return filepath.ToSlash(clean), physical, nil
}

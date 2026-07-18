package workspace

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var safeOverrideName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

type ProjectContext struct {
	ProjectDir    string
	ProjectName   string
	Slug          string
	MountDir      string
	ContainerName string
	HostAlias     string
	MarkerAlias   string
	ProjectHash   string
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func ResolveProjectContext(projectDir, containerName, hostAlias string) (ProjectContext, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("resolve project dir: %w", err)
	}
	base := filepath.Base(absDir)
	sum := sha1.Sum([]byte(absDir))
	shortHash := hex.EncodeToString(sum[:])[:8]
	slug := SanitizeName(base)
	if slug == "" {
		slug = "workspace-" + shortHash
	}
	if len(slug) > 63 {
		slug = strings.TrimRight(slug[:63], "-")
	}
	mountName := base
	if mountName == "." || mountName == string(filepath.Separator) || mountName == "" {
		mountName = slug
	}

	resolvedContainerName := strings.TrimSpace(containerName)
	if resolvedContainerName == "" {
		resolvedContainerName = "elyro-workspace-" + slug + "-" + shortHash
	} else if !safeOverrideName.MatchString(resolvedContainerName) {
		return ProjectContext{}, fmt.Errorf("container name %q must contain only letters, numbers, dot, underscore, and hyphen", resolvedContainerName)
	}

	resolvedHostAlias := strings.TrimSpace(hostAlias)
	if resolvedHostAlias == "" {
		resolvedHostAlias = "elyro-" + slug + "-" + shortHash
	} else if !safeOverrideName.MatchString(resolvedHostAlias) {
		return ProjectContext{}, fmt.Errorf("SSH host alias %q must contain only letters, numbers, dot, underscore, and hyphen", resolvedHostAlias)
	}

	return ProjectContext{
		ProjectDir:    absDir,
		ProjectName:   base,
		Slug:          slug,
		MountDir:      path.Join("/home/elyro", mountName),
		ContainerName: resolvedContainerName,
		HostAlias:     resolvedHostAlias,
		MarkerAlias:   resolvedHostAlias,
		ProjectHash:   shortHash,
	}, nil
}

func SanitizeName(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == '.' || r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

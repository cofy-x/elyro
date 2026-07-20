package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	KindWorkspace = "workspace"
)

type Store struct {
	SchemaVersion int      `json:"schema_version"`
	Workspaces    []Record `json:"workspaces"`
}

type Record struct {
	Name                  string    `json:"name"`
	Kind                  string    `json:"kind"`
	ProjectDir            string    `json:"project_dir"`
	HostWorkspaceDir      string    `json:"host_workspace_dir"`
	ContainerWorkspaceDir string    `json:"container_workspace_dir"`
	ContainerName         string    `json:"container_name"`
	Hostname              string    `json:"hostname"`
	SSHAlias              string    `json:"ssh_alias"`
	Environment           string    `json:"environment"`
	Toolchain             string    `json:"toolchain"`
	Platform              string    `json:"platform"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func DefaultPath() (string, error) {
	if stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); stateHome != "" {
		return filepath.Join(stateHome, "elyro", "workspaces.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "elyro", "workspaces.json"), nil
}

func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{SchemaVersion: 1}, nil
		}
		return Store{}, fmt.Errorf("read workspace registry: %w", err)
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return Store{}, fmt.Errorf("parse workspace registry: %w", err)
	}
	if store.SchemaVersion != 1 {
		return Store{}, fmt.Errorf("unsupported workspace registry schema %d in %s (supported: 1)", store.SchemaVersion, path)
	}
	return store, nil
}

func Save(path string, store Store) error {
	if store.SchemaVersion == 0 {
		store.SchemaVersion = 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create workspace registry dir: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace registry: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".workspaces-*.json")
	if err != nil {
		return fmt.Errorf("create workspace registry temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write workspace registry temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod workspace registry temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close workspace registry temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace workspace registry: %w", err)
	}
	return nil
}

func Upsert(store Store, record Record) (Store, error) {
	normalized, err := normalizeRecord(record)
	if err != nil {
		return Store{}, err
	}
	if store.SchemaVersion == 0 {
		store.SchemaVersion = 1
	}
	replaced := false
	for i := range store.Workspaces {
		item := store.Workspaces[i]
		if item.Kind == normalized.Kind && physicalPath(item.ProjectDir) == physicalPath(normalized.ProjectDir) {
			store.Workspaces[i] = normalized
			replaced = true
			break
		}
	}
	if !replaced {
		store.Workspaces = append(store.Workspaces, normalized)
	}
	Sort(store.Workspaces)
	return store, nil
}

func UpsertFile(path string, record Record) error {
	store, err := Load(path)
	if err != nil {
		return err
	}
	store, err = Upsert(store, record)
	if err != nil {
		return err
	}
	return Save(path, store)
}

func RemoveFile(path, projectDir string) error {
	store, err := Load(path)
	if err != nil {
		return err
	}
	projectDir, err = cleanAbs(projectDir)
	if err != nil {
		return fmt.Errorf("project_dir: %w", err)
	}
	workspaces := store.Workspaces[:0]
	removed := false
	for _, record := range store.Workspaces {
		if record.Kind == KindWorkspace && physicalPath(record.ProjectDir) == physicalPath(projectDir) {
			removed = true
			continue
		}
		workspaces = append(workspaces, record)
	}
	if !removed {
		return nil
	}
	store.Workspaces = workspaces
	return Save(path, store)
}

func HasWorkspaceRecord(store Store, projectDir string) (bool, error) {
	projectDir, err := cleanAbs(projectDir)
	if err != nil {
		return false, fmt.Errorf("project_dir: %w", err)
	}
	for _, record := range store.Workspaces {
		if record.Kind == KindWorkspace && physicalPath(record.ProjectDir) == physicalPath(projectDir) {
			return true, nil
		}
	}
	return false, nil
}

func Current(store Store, cwd string) (Record, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Record{}, fmt.Errorf("resolve current directory: %w", err)
	}
	abs = physicalPath(abs)
	var matches []Record
	for _, record := range store.Workspaces {
		if record.Kind != KindWorkspace {
			continue
		}
		if pathContains(record.HostWorkspaceDir, abs) {
			matches = append(matches, record)
		}
	}
	if len(matches) == 0 {
		return Record{}, ErrNoCurrent
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return len(matches[i].HostWorkspaceDir) > len(matches[j].HostWorkspaceDir)
	})
	return matches[0], nil
}

func FindWorkspaceByName(store Store, name string) (Record, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return Record{}, errors.New("workspace name is required")
	}
	var matches []Record
	for _, record := range store.Workspaces {
		if record.Kind == KindWorkspace && record.Name == target {
			matches = append(matches, record)
		}
	}
	switch len(matches) {
	case 0:
		return Record{}, fmt.Errorf("workspace %q not found", target)
	case 1:
		return matches[0], nil
	default:
		return Record{}, fmt.Errorf("workspace name %q is ambiguous", target)
	}
}

func Sort(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Kind != records[j].Kind {
			return records[i].Kind < records[j].Kind
		}
		return records[i].Name < records[j].Name
	})
}

var ErrNoCurrent = errors.New("no current workspace")

func normalizeRecord(record Record) (Record, error) {
	record.Name = strings.TrimSpace(record.Name)
	record.Kind = strings.TrimSpace(record.Kind)
	if record.Kind == "" {
		record.Kind = KindWorkspace
	}
	if record.Name == "" {
		return Record{}, errors.New("workspace name is required")
	}
	projectDir, err := cleanAbs(record.ProjectDir)
	if err != nil {
		return Record{}, fmt.Errorf("project_dir: %w", err)
	}
	hostWorkspaceDir, err := cleanAbs(firstNonEmpty(record.HostWorkspaceDir, record.ProjectDir))
	if err != nil {
		return Record{}, fmt.Errorf("host_workspace_dir: %w", err)
	}
	record.ProjectDir = projectDir
	record.HostWorkspaceDir = hostWorkspaceDir
	record.ContainerWorkspaceDir = strings.TrimSpace(record.ContainerWorkspaceDir)
	record.ContainerName = strings.TrimSpace(record.ContainerName)
	record.Hostname = strings.TrimSpace(record.Hostname)
	record.SSHAlias = strings.TrimSpace(record.SSHAlias)
	record.Environment = strings.TrimSpace(record.Environment)
	record.Toolchain = strings.TrimSpace(record.Toolchain)
	record.Platform = strings.TrimSpace(record.Platform)
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}
	return record, nil
}

func cleanAbs(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func pathContains(parent, child string) bool {
	parent = physicalPath(parent)
	child = physicalPath(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func physicalPath(path string) string {
	cleaned := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return cleaned
	}
	return filepath.Clean(resolved)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

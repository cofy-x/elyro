package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpsertAndLoadSaveRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "elyro", "workspaces.json")
	record := Record{
		Name:                  "demo",
		Kind:                  KindWorkspace,
		ProjectDir:            filepath.Join(t.TempDir(), "demo"),
		HostWorkspaceDir:      filepath.Join(t.TempDir(), "demo"),
		ContainerWorkspaceDir: "/home/elyro/demo",
		ContainerName:         "elyro-workspace-demo",
		SSHAlias:              "elyro-demo",
		Environment:           "python",
		Toolchain:             "python",
		Platform:              "linux/amd64",
		UpdatedAt:             time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}
	if err := UpsertFile(path, record); err != nil {
		t.Fatalf("UpsertFile() error = %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := len(loaded.Workspaces); got != 1 {
		t.Fatalf("workspace count = %d, want 1", got)
	}
	if loaded.Workspaces[0].Name != "demo" || loaded.Workspaces[0].SSHAlias != "elyro-demo" {
		t.Fatalf("loaded workspace = %+v, want demo/elyro-demo", loaded.Workspaces[0])
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat registry: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("registry mode = %o, want 644", info.Mode().Perm())
	}
}

func TestLoadRejectsUnsupportedRegistrySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	if err := os.WriteFile(path, []byte("{\"schema_version\":2,\"workspaces\":[]}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported workspace registry schema 2") {
		t.Fatalf("Load() error = %v, want unsupported schema diagnostic", err)
	}
}

func TestRemoveFileDeletesOnlyMatchingProject(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state", "elyro", "workspaces.json")
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, record := range []Record{
		{Name: "first", Kind: KindWorkspace, ProjectDir: first, HostWorkspaceDir: first},
		{Name: "second", Kind: KindWorkspace, ProjectDir: second, HostWorkspaceDir: second},
	} {
		if err := UpsertFile(path, record); err != nil {
			t.Fatal(err)
		}
	}

	if err := RemoveFile(path, first); err != nil {
		t.Fatal(err)
	}
	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Workspaces) != 1 || store.Workspaces[0].Name != "second" {
		t.Fatalf("workspaces after removal = %+v, want only second", store.Workspaces)
	}
}

func TestRemoveFileDoesNotCreateMissingRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "elyro", "workspaces.json")
	if err := RemoveFile(path, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("registry stat error = %v, want not exist", err)
	}
}

func TestCurrentSelectsNearestParentWorkspace(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	nested := filepath.Join(child, "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	store := Store{SchemaVersion: 1, Workspaces: []Record{
		{Name: "parent", Kind: KindWorkspace, ProjectDir: parent, HostWorkspaceDir: parent},
		{Name: "child", Kind: KindWorkspace, ProjectDir: child, HostWorkspaceDir: child},
	}}
	got, err := Current(store, nested)
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if got.Name != "child" {
		t.Fatalf("Current() = %q, want child", got.Name)
	}
}

func TestCurrentMissing(t *testing.T) {
	store := Store{SchemaVersion: 1, Workspaces: []Record{
		{Name: "other", Kind: KindWorkspace, ProjectDir: t.TempDir(), HostWorkspaceDir: t.TempDir()},
	}}
	_, err := Current(store, t.TempDir())
	if !errors.Is(err, ErrNoCurrent) {
		t.Fatalf("Current() error = %v, want ErrNoCurrent", err)
	}
}

func TestFindWorkspaceByName(t *testing.T) {
	store := Store{SchemaVersion: 1, Workspaces: []Record{
		{Name: "demo", Kind: KindWorkspace, ProjectDir: "/tmp/demo", HostWorkspaceDir: "/tmp/demo"},
	}}
	got, err := FindWorkspaceByName(store, "demo")
	if err != nil {
		t.Fatalf("FindWorkspaceByName() error = %v", err)
	}
	if got.Name != "demo" {
		t.Fatalf("FindWorkspaceByName() = %q, want demo", got.Name)
	}
}

func TestFindWorkspaceByNameReportsAmbiguousNames(t *testing.T) {
	store := Store{SchemaVersion: 1, Workspaces: []Record{
		{Name: "demo", Kind: KindWorkspace, ProjectDir: "/tmp/one", HostWorkspaceDir: "/tmp/one"},
		{Name: "demo", Kind: KindWorkspace, ProjectDir: "/tmp/two", HostWorkspaceDir: "/tmp/two"},
	}}
	_, err := FindWorkspaceByName(store, "demo")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("FindWorkspaceByName() error = %v, want ambiguous", err)
	}
}

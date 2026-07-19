package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectRootPrecedence(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	configured := filepath.Join(repo, "services", "api")
	nested := filepath.Join(configured, "internal", "http")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configured, defaultEnvironmentConfigFile)
	if err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	storePath, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := UpsertFile(storePath, Record{Name: "repo", ProjectDir: repo, HostWorkspaceDir: repo}); err != nil {
		t.Fatal(err)
	}

	root, err := ResolveProjectRoot(nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if root.Dir != physicalPath(configured) || root.Source != ProjectRootSourceConfig || root.ConfigPath != filepath.Join(physicalPath(configured), defaultEnvironmentConfigFile) {
		t.Fatalf("root = %#v, want configured project", root)
	}

	root, err = ResolveProjectRoot(nested, true)
	if err != nil {
		t.Fatal(err)
	}
	if root.Dir != filepath.Clean(nested) || root.Source != ProjectRootSourceExplicit {
		t.Fatalf("explicit root = %#v, want nested", root)
	}
}

func TestResolveProjectRootUsesRegistryBeforeGit(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	nested := filepath.Join(repo, "pkg", "api")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	workspaceDir := filepath.Join(repo, "pkg")
	storePath, _ := DefaultPath()
	if err := UpsertFile(storePath, Record{Name: "pkg", ProjectDir: workspaceDir, HostWorkspaceDir: workspaceDir}); err != nil {
		t.Fatal(err)
	}
	root, err := ResolveProjectRoot(nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if root.Dir != filepath.Clean(workspaceDir) || root.Source != ProjectRootSourceRegistry {
		t.Fatalf("root = %#v, want registry workspace", root)
	}
}

func TestResolveProjectRootRecognizesGitFileAndSymlink(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repo, "src", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "project")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	root, err := ResolveProjectRoot(filepath.Join(link, "src", "pkg"), false)
	if err != nil {
		t.Fatal(err)
	}
	if root.Dir != physicalPath(repo) || root.Source != ProjectRootSourceGit {
		t.Fatalf("root = %#v, want git worktree root", root)
	}
}

func TestResolveProjectRootFallsBackToCurrentDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	root, err := ResolveProjectRoot(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if root.Dir != physicalPath(dir) || root.Source != ProjectRootSourceCWD {
		t.Fatalf("root = %#v, want cwd", root)
	}
}

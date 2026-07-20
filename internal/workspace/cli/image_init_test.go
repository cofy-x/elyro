package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProjectImageCreatesConfigAndDockerfile(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	var out bytes.Buffer
	err := initProjectImage(imageInitOptions{
		ProjectDir: project, Toolchain: "go", Image: "elyro-local/demo:dev", Yes: true,
		In: strings.NewReader(""), Out: &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	config := readTestFile(t, filepath.Join(project, "elyro.yaml"))
	for _, want := range []string{"toolchain: go", "image: elyro-local/demo:dev", "context: .", "dockerfile: .elyro/Dockerfile"} {
		if !strings.Contains(config, want) {
			t.Errorf("config missing %q:\n%s", want, config)
		}
	}
	dockerfile := readTestFile(t, filepath.Join(project, ".elyro", "Dockerfile"))
	if !strings.Contains(dockerfile, "workspace-go:") || strings.Contains(dockerfile, "apt-get install -y --no-install-recommends sqlite3\n") {
		t.Fatalf("unexpected Dockerfile:\n%s", dockerfile)
	}
	if !strings.Contains(out.String(), "elyro image build") {
		t.Fatalf("receipt = %q", out.String())
	}
}

func TestInitProjectImagePreservesExistingConfigComments(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	config := "# project note\nversion: 1\ndefault_environment: dev\nenvironments:\n  dev: # keep me\n    toolchain: go\n    docker:\n      publish:\n        - 8080\n  release:\n    toolchain: node\n"
	if err := os.WriteFile(filepath.Join(project, "elyro.yaml"), []byte(config), 0o640); err != nil {
		t.Fatal(err)
	}
	err := initProjectImage(imageInitOptions{
		ProjectDir: project, Image: "elyro-local/demo:dev", Yes: true,
		In: strings.NewReader(""), Out: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated := readTestFile(t, filepath.Join(project, "elyro.yaml"))
	for _, want := range []string{"# project note", "# keep me", "release:", "publish:", "image: elyro-local/demo:dev", "build:"} {
		if !strings.Contains(updated, want) {
			t.Errorf("updated config missing %q:\n%s", want, updated)
		}
	}
	info, err := os.Stat(filepath.Join(project, "elyro.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestInitProjectImageFailureLeavesNoPartialFiles(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, ".elyro")); err != nil {
		t.Fatal(err)
	}
	err := initProjectImage(imageInitOptions{
		ProjectDir: project, Toolchain: "go", Image: "elyro-local/demo:dev", Yes: true,
		In: strings.NewReader(""), Out: &bytes.Buffer{},
	})
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "elyro.yaml")); !os.IsNotExist(err) {
		t.Fatalf("elyro.yaml should not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "Dockerfile")); !os.IsNotExist(err) {
		t.Fatalf("outside Dockerfile should not exist: %v", err)
	}
}

func TestInitProjectImageRejectsSymlinkedConfig(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	target := filepath.Join(t.TempDir(), "elyro.yaml")
	if err := os.WriteFile(target, []byte("version: 1\nenvironments: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(project, "elyro.yaml")); err != nil {
		t.Fatal(err)
	}
	err := initProjectImage(imageInitOptions{ProjectDir: project, Toolchain: "go", Image: "elyro-local/demo:dev", Yes: true, Out: &bytes.Buffer{}})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("error = %v", err)
	}
	if got := readTestFile(t, target); got != "version: 1\nenvironments: {}\n" {
		t.Fatalf("symlink target changed: %q", got)
	}
}

func TestInitProjectImageNonInteractiveRequiresImage(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	err := initProjectImage(imageInitOptions{ProjectDir: project, Toolchain: "go", Yes: true, Out: &bytes.Buffer{}})
	if err == nil || !strings.Contains(err.Error(), "--image") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitProjectImageExplicitEnvironmentPreservesDefaultAndReusesImage(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	config := "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n  tools:\n    toolchain: node\n    image: elyro-local/tools:dev\n"
	if err := os.WriteFile(filepath.Join(project, "elyro.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := initProjectImage(imageInitOptions{
		ProjectDir: project, Environment: "tools", Yes: true,
		In: strings.NewReader(""), Out: &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated := readTestFile(t, filepath.Join(project, "elyro.yaml"))
	if !strings.Contains(updated, "default_environment: dev") || !strings.Contains(updated, "image: elyro-local/tools:dev") || !strings.Contains(updated, "build:") {
		t.Fatalf("updated config:\n%s", updated)
	}
	if !strings.Contains(out.String(), `elyro image build --environment "tools"`) {
		t.Fatalf("receipt = %q", out.String())
	}
}

func TestInitProjectImageRejectsConflictsWithoutWrites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, config, image, want string
	}{
		{"conflicting image", "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    image: elyro-local/existing:dev\n", "elyro-local/other:dev", "already uses image"},
		{"existing build", "version: 1\ndefault_environment: dev\nenvironments:\n  dev:\n    toolchain: go\n    image: elyro-local/existing:dev\n    build:\n      context: .\n      dockerfile: Dockerfile\n", "", "already has image build"},
	} {
		t.Run(test.name, func(t *testing.T) {
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, "elyro.yaml"), []byte(test.config), 0o644); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(test.config, "dockerfile: Dockerfile") {
				if err := os.WriteFile(filepath.Join(project, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before := readTestFile(t, filepath.Join(project, "elyro.yaml"))
			err := initProjectImage(imageInitOptions{ProjectDir: project, Image: test.image, Yes: true, Out: &bytes.Buffer{}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if after := readTestFile(t, filepath.Join(project, "elyro.yaml")); after != before {
				t.Fatalf("config changed after failure:\n%s", after)
			}
			if _, err := os.Stat(filepath.Join(project, ".elyro", "Dockerfile")); !os.IsNotExist(err) {
				t.Fatalf("generated Dockerfile exists after failure: %v", err)
			}
		})
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

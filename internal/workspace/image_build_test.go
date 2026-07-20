package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateBuildImageReference(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, image, wantError string
	}{
		{"tagged", "elyro-local/demo:dev", ""},
		{"registry port", "localhost:5000/team/demo:v1", ""},
		{"missing tag", "elyro-local/demo", "explicit tag"},
		{"latest", "elyro-local/demo:latest", "latest"},
		{"digest", "elyro-local/demo@sha256:" + strings.Repeat("a", 64), "digest"},
		{"tag and digest", "elyro-local/demo:dev@sha256:" + strings.Repeat("a", 64), "digest"},
		{"invalid", "Bad Image:dev", "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateBuildImageReference(test.image)
			if test.wantError == "" && err != nil {
				t.Fatalf("ValidateBuildImageReference() error = %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("ValidateBuildImageReference() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestResolveImageBuildPaths(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".elyro"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".elyro", "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	build, err := ResolveImageBuild(project, "elyro-local/demo:dev", ".", ".elyro/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	if build.Context != "." || build.Dockerfile != ".elyro/Dockerfile" {
		t.Fatalf("build = %#v", build)
	}
}

func TestResolveImageBuildRejectsUnsafePaths(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, ".elyro")); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ context, dockerfile, want string }{
		{"/tmp", ".elyro/Dockerfile", "project-relative"},
		{"..", ".elyro/Dockerfile", "escapes"},
		{".", ".elyro/Dockerfile", "outside"},
		{".", "missing", "no such file"},
	} {
		_, err := ResolveImageBuild(project, "elyro-local/demo:dev", test.context, test.dockerfile)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
			t.Errorf("ResolveImageBuild(%q, %q) error = %v, want %q", test.context, test.dockerfile, err, test.want)
		}
	}
}

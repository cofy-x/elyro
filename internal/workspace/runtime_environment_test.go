package workspace

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveRuntimeEnvironmentAppliesPrecedenceAndStableDigest(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	writeRuntimeEnvironmentFile(t, projectDir, ".elyro/dev.env", "SHARED=file-one\r\nFILE_ONLY=first\r\nEMPTY=\r\nHASH=a#b\r\nLITERAL='${HOME}'\r\n")
	writeRuntimeEnvironmentFile(t, projectDir, ".elyro/local.env", "SHARED=file-two\nLOCAL_ONLY=value=with=equals\n")

	got, err := ResolveRuntimeEnvironment(projectDir, map[string]string{"SHARED": "inline", "INLINE_ONLY": "yes"}, []string{".elyro/dev.env", ".elyro/local.env"})
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"EMPTY", "FILE_ONLY", "HASH", "INLINE_ONLY", "LITERAL", "LOCAL_ONLY", "SHARED"}
	if !slices.Equal(got.VariableNames, wantNames) {
		t.Fatalf("variable names = %v, want %v", got.VariableNames, wantNames)
	}
	if got.Digest == "" {
		t.Fatal("non-empty runtime environment has empty digest")
	}
	if !slices.Equal(DockerRuntimeEnvironmentArgs(got), []string{
		"--env-file", got.EnvFiles[0].PhysicalPath,
		"--env-file", got.EnvFiles[1].PhysicalPath,
		"--env", "INLINE_ONLY=yes", "--env", "SHARED=inline",
	}) {
		t.Fatalf("DockerRuntimeEnvironmentArgs() = %#v", DockerRuntimeEnvironmentArgs(got))
	}

	writeRuntimeEnvironmentFile(t, projectDir, ".elyro/equivalent.env", "EMPTY=\nFILE_ONLY=first\nHASH=a#b\nLITERAL='${HOME}'\nLOCAL_ONLY=value=with=equals\nSHARED=inline\nINLINE_ONLY=yes\n")
	equivalent, err := ResolveRuntimeEnvironment(projectDir, nil, []string{".elyro/equivalent.env"})
	if err != nil {
		t.Fatal(err)
	}
	if equivalent.Digest != got.Digest {
		t.Fatalf("equivalent effective values changed digest: %q != %q", equivalent.Digest, got.Digest)
	}

	changed, err := ResolveRuntimeEnvironment(projectDir, map[string]string{"SHARED": "changed"}, []string{".elyro/dev.env", ".elyro/local.env"})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Digest == got.Digest {
		t.Fatal("changed effective values retained digest")
	}
}

func TestResolveRuntimeEnvironmentRejectsInvalidFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "bare key", content: "HOST_VALUE\n", want: "bare variables"},
		{name: "duplicate", content: "VALUE=one\nVALUE=two\n", want: "duplicates variable"},
		{name: "invalid name", content: "BAD-NAME=value\n", want: "invalid environment variable name"},
		{name: "export", content: "export VALUE=one\n", want: "invalid environment variable name"},
		{name: "embedded carriage return", content: "VALUE=one\rtwo\n", want: "unsupported carriage return"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projectDir := t.TempDir()
			writeRuntimeEnvironmentFile(t, projectDir, "bad.env", test.content)
			_, err := ResolveRuntimeEnvironment(projectDir, nil, []string{"bad.env"})
			if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), `"bad.env" line `) {
				t.Fatalf("error = %v, want path, line, and %q", err, test.want)
			}
		})
	}
}

func TestResolveRuntimeEnvironmentRejectsInvalidInlineValues(t *testing.T) {
	t.Parallel()
	for name, values := range map[string]map[string]string{
		"invalid name": {"BAD-NAME": "value"},
		"newline":      {"VALUE": "first\nsecond"},
		"nul":          {"VALUE": "first\x00second"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ResolveRuntimeEnvironment(t.TempDir(), values, nil)
			if err == nil {
				t.Fatal("ResolveRuntimeEnvironment accepted invalid inline value")
			}
		})
	}
}

func TestResolveRuntimeEnvironmentRejectsUnsafePaths(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	writeRuntimeEnvironmentFile(t, projectDir, "valid.env", "VALUE=one\n")
	if err := os.Mkdir(filepath.Join(projectDir, "directory.env"), 0o755); err != nil {
		t.Fatal(err)
	}
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.env")
	if err := os.WriteFile(outsideFile, []byte("VALUE=outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(projectDir, "escape.env")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(projectDir, "valid.env"), filepath.Join(projectDir, "valid-alias.env")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		paths []string
	}{
		{name: "absolute", paths: []string{outsideFile}},
		{name: "parent", paths: []string{"../outside.env"}},
		{name: "missing", paths: []string{"missing.env"}},
		{name: "directory", paths: []string{"directory.env"}},
		{name: "symlink escape", paths: []string{"escape.env"}},
		{name: "duplicate", paths: []string{"valid.env", "./valid.env"}},
		{name: "duplicate symlink target", paths: []string{"valid.env", "valid-alias.env"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveRuntimeEnvironment(projectDir, nil, test.paths); err == nil {
				t.Fatalf("accepted unsafe paths %#v", test.paths)
			}
		})
	}
}

func TestResolveRuntimeEnvironmentSupportsSymlinkedProjectRoot(t *testing.T) {
	t.Parallel()
	realProject := t.TempDir()
	writeRuntimeEnvironmentFile(t, realProject, ".elyro/dev.env", "VALUE=one\n")
	linkParent := t.TempDir()
	linkedProject := filepath.Join(linkParent, "project")
	if err := os.Symlink(realProject, linkedProject); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRuntimeEnvironment(linkedProject, nil, []string{".elyro/dev.env"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest == "" || len(got.EnvFiles) != 1 {
		t.Fatalf("resolved runtime environment = %#v", got)
	}
}

func TestEmptyRuntimeEnvironmentHasLegacyEmptyDigest(t *testing.T) {
	t.Parallel()
	got, err := ResolveRuntimeEnvironment(t.TempDir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest != "" || len(got.VariableNames) != 0 || len(DockerRuntimeEnvironmentArgs(got)) != 0 {
		t.Fatalf("empty runtime environment = %#v", got)
	}
}

func writeRuntimeEnvironmentFile(t *testing.T, projectDir, relative, content string) {
	t.Helper()
	path := filepath.Join(projectDir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

package workspace

import (
	"strings"
	"testing"
)

func TestResolveProjectContextDefaults(t *testing.T) {
	t.Parallel()

	ctx, err := ResolveProjectContext("/tmp/My Sample.Project", "", "")
	if err != nil {
		t.Fatalf("ResolveProjectContext returned error: %v", err)
	}

	if got, want := ctx.Slug, "my-sample-project"; got != want {
		t.Fatalf("slug mismatch: got %q want %q", got, want)
	}
	if got, want := ctx.MountDir, "/home/elyro/My Sample.Project"; got != want {
		t.Fatalf("mount dir mismatch: got %q want %q", got, want)
	}
	if got, prefix := ctx.ContainerName, "elyro-workspace-my-sample-project-"; len(got) != len(prefix)+8 || got[:len(prefix)] != prefix {
		t.Fatalf("container mismatch: got %q want prefix %q plus hash", got, prefix)
	}
	if got, prefix := ctx.HostAlias, "elyro-my-sample-project-"; len(got) != len(prefix)+8 || got[:len(prefix)] != prefix {
		t.Fatalf("host alias mismatch: got %q want prefix %q plus hash", got, prefix)
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"demo":                "demo",
		" Demo Workspace ":    "demo-workspace",
		"hello_world.v1":      "hello-world-v1",
		"***":                 "",
		"中文项目":                "",
		"snake case.and-dash": "snake-case-and-dash",
	}

	for input, want := range tests {
		if got := SanitizeName(input); got != want {
			t.Fatalf("SanitizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveProjectContextProducesDockerSafeHostname(t *testing.T) {
	t.Parallel()

	unicodeProject, err := ResolveProjectContext("/tmp/中文项目", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(unicodeProject.Slug) > 63 || !strings.HasPrefix(unicodeProject.Slug, "workspace-") {
		t.Fatalf("unicode project slug = %q", unicodeProject.Slug)
	}

	longProject, err := ResolveProjectContext("/tmp/"+strings.Repeat("long-name-", 20), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(longProject.Slug) > 63 || strings.HasSuffix(longProject.Slug, "-") {
		t.Fatalf("long project slug = %q (len %d)", longProject.Slug, len(longProject.Slug))
	}
}

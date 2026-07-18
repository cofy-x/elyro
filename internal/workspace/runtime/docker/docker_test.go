package docker

import "testing"

func TestParseInspectOutputKeepsEmptyPublishedField(t *testing.T) {
	output := "id123\t/elyro-workspace-demo\telyro/workspace-python:latest-amd64\trunning\tdemo\tpython\tpython\telyro/workspace-python:latest-amd64\tlinux/amd64\t./tmp/demo\telyro-demo\t49123\t\tfalse\t\n"

	got, err := parseInspectOutput("id123", output)
	if err != nil {
		t.Fatal(err)
	}
	if got.Published != "" {
		t.Fatalf("expected empty published field, got %q", got.Published)
	}
	if got.Hostname != "demo" {
		t.Fatalf("expected hostname demo, got %q", got.Hostname)
	}
	if got.Privileged != "false" {
		t.Fatalf("expected privileged field %q, got %q", "false", got.Privileged)
	}
	if got.Mounts != "" {
		t.Fatalf("expected empty mounts field, got %q", got.Mounts)
	}
}

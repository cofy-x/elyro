package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTransientSSHScanError(t *testing.T) {
	t.Parallel()

	for _, message := range []string{
		"scan workspace SSH host key: exit status 1: Connection closed by remote host",
		"scan workspace SSH host key: no keys returned for 127.0.0.1:2222",
		"scan workspace SSH host key: exit status 1: Connection refused",
	} {
		if !isTransientSSHScanError(errors.New(message)) {
			t.Fatalf("expected transient error for %q", message)
		}
	}
	if isTransientSSHScanError(errors.New("exec: ssh-keyscan: executable file not found")) {
		t.Fatal("missing ssh-keyscan must fail without retry")
	}
}

func TestKnownHostBlockRoundTrip(t *testing.T) {
	block := buildKnownHostBlock("elyro-demo", "container-1", "[127.0.0.1]:2222 ssh-ed25519 AAAA")
	id, keys := knownHostBlock("unmanaged\n"+block, "elyro-demo")
	if id != "container-1" || keys != "[127.0.0.1]:2222 ssh-ed25519 AAAA" {
		t.Fatalf("knownHostBlock = (%q, %q)", id, keys)
	}
	removed := removeKnownHostBlock("unmanaged\n"+block, "elyro-demo")
	if strings.Contains(removed, "ELYRO_WORKSPACE_KNOWN_HOST") || !strings.Contains(removed, "unmanaged") {
		t.Fatalf("removeKnownHostBlock = %q", removed)
	}
}

func TestPrepareKnownSSHHostLifecycle(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "known_hosts")
	keys := "[127.0.0.1]:2222 ssh-ed25519 AAAA"
	scan := func(context.Context, string, string) (string, error) { return keys, nil }
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
		t.Fatal(err)
	}
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
		t.Fatalf("idempotent registration failed: %v", err)
	}

	keys = "[127.0.0.1]:2222 ssh-ed25519 CHANGED"
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err == nil || !strings.Contains(err.Error(), "changed unexpectedly") {
		t.Fatalf("same-container key change error = %v", err)
	}
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-2", "127.0.0.1", "2222", scan); err != nil {
		t.Fatalf("replacement container refresh failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "container-1") || !strings.Contains(string(content), "container-2") || !strings.Contains(string(content), "CHANGED") {
		t.Fatalf("refreshed known-hosts = %q", content)
	}
	if err := RemoveKnownSSHHost(path, "elyro-demo"); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "elyro-demo") {
		t.Fatalf("removed known-hosts = %q", content)
	}
}

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
	if err := ValidateKnownSSHHost(path, "elyro-demo", "container-1"); err != nil {
		t.Fatalf("ValidateKnownSSHHost returned error: %v", err)
	}
	if err := ValidateKnownSSHHost(path, "elyro-demo", "other-container"); err == nil {
		t.Fatal("ValidateKnownSSHHost accepted a mismatched container")
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

func TestPrepareKnownSSHHostMergesCompatiblePartialScans(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "known_hosts")
	keys := "[127.0.0.1]:2222 ssh-ed25519 ED25519"
	scan := func(context.Context, string, string) (string, error) { return keys, nil }
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
		t.Fatal(err)
	}

	keys = strings.Join([]string{
		"[127.0.0.1]:2222 ssh-ed25519 ED25519",
		"[127.0.0.1]:2222 ssh-rsa RSA",
	}, "\n")
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
		t.Fatalf("compatible expanded scan failed: %v", err)
	}

	keys = "[127.0.0.1]:2222 ssh-rsa RSA"
	if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
		t.Fatalf("compatible reduced scan failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "ED25519") || !strings.Contains(string(content), "RSA") {
		t.Fatalf("merged known-hosts = %q", content)
	}
}

func TestPrepareKnownSSHHostRejectsChangedOrUnrelatedPartialScan(t *testing.T) {
	t.Parallel()

	for _, changed := range []string{
		"[127.0.0.1]:2222 ssh-ed25519 CHANGED",
		"[127.0.0.1]:2222 ssh-rsa UNRELATED",
	} {
		path := filepath.Join(t.TempDir(), "known_hosts")
		keys := "[127.0.0.1]:2222 ssh-ed25519 ORIGINAL"
		scan := func(context.Context, string, string) (string, error) { return keys, nil }
		if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err != nil {
			t.Fatal(err)
		}
		keys = changed
		if err := prepareKnownSSHHost(t.Context(), path, "elyro-demo", "container-1", "127.0.0.1", "2222", scan); err == nil || !strings.Contains(err.Error(), "changed unexpectedly") {
			t.Fatalf("changed scan %q error = %v", changed, err)
		}
	}
}

func TestMergeKnownSSHKeysPreservesIdenticalLegacyContent(t *testing.T) {
	t.Parallel()

	const keys = "legacy-known-host-content"
	merged, ok := mergeKnownSSHKeys(keys, keys)
	if !ok || merged != keys {
		t.Fatalf("mergeKnownSSHKeys() = (%q, %t)", merged, ok)
	}
	if _, ok := mergeKnownSSHKeys(keys, "different-legacy-content"); ok {
		t.Fatal("mergeKnownSSHKeys accepted different unstructured content")
	}
}

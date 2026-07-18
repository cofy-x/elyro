package access

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSSHIdentityGeneratesOpenSSHKey(t *testing.T) {
	t.Parallel()

	identityFile := filepath.Join(t.TempDir(), "elyro_workspace_ed25519")
	resolved, publicKey, err := EnsureSSHIdentity(identityFile)
	if err != nil {
		t.Fatalf("EnsureSSHIdentity returned error: %v", err)
	}
	if resolved != identityFile {
		t.Fatalf("resolved identity mismatch: got %q want %q", resolved, identityFile)
	}
	if !strings.HasPrefix(publicKey, "ssh-ed25519 ") {
		t.Fatalf("unexpected public key: %q", publicKey)
	}

	privateData, err := os.ReadFile(identityFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(privateData), "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatalf("private key is not in OpenSSH format:\n%s", string(privateData))
	}

	publicData, err := os.ReadFile(identityFile + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(publicData)) != publicKey {
		t.Fatalf("public key file mismatch: got %q want %q", strings.TrimSpace(string(publicData)), publicKey)
	}
}

func TestEnsureSSHIdentityDerivesMissingPublicKey(t *testing.T) {
	t.Parallel()

	identityFile := filepath.Join(t.TempDir(), "elyro_workspace_ed25519")
	_, originalPublicKey, err := EnsureSSHIdentity(identityFile)
	if err != nil {
		t.Fatalf("EnsureSSHIdentity returned error: %v", err)
	}
	if err := os.Remove(identityFile + ".pub"); err != nil {
		t.Fatal(err)
	}

	_, derivedPublicKey, err := EnsureSSHIdentity(identityFile)
	if err != nil {
		t.Fatalf("EnsureSSHIdentity returned error after removing public key: %v", err)
	}
	if derivedPublicKey != originalPublicKey {
		t.Fatalf("derived public key mismatch: got %q want %q", derivedPublicKey, originalPublicKey)
	}
}

func TestEnsureSSHIdentityRewritesStalePublicKey(t *testing.T) {
	t.Parallel()

	identityFile := filepath.Join(t.TempDir(), "elyro_workspace_ed25519")
	_, originalPublicKey, err := EnsureSSHIdentity(identityFile)
	if err != nil {
		t.Fatalf("EnsureSSHIdentity returned error: %v", err)
	}
	if err := os.WriteFile(identityFile+".pub", []byte("ssh-ed25519 stale-key elyro-workspace\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, publicKey, err := EnsureSSHIdentity(identityFile)
	if err != nil {
		t.Fatalf("EnsureSSHIdentity returned error with stale public key: %v", err)
	}
	if publicKey != originalPublicKey {
		t.Fatalf("public key mismatch: got %q want %q", publicKey, originalPublicKey)
	}

	publicData, err := os.ReadFile(identityFile + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(publicData)) != originalPublicKey {
		t.Fatalf("public key file was not rewritten: got %q want %q", strings.TrimSpace(string(publicData)), originalPublicKey)
	}
}

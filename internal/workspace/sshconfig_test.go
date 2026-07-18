package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertManagedSSHHost(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("Host existing\n  HostName 1.2.3.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	entry := SSHHostEntry{
		HostAlias:      "elyro-demo",
		HostName:       "127.0.0.1",
		Port:           "2200",
		User:           "elyro",
		IdentityFile:   "/tmp/elyro-workspace-test",
		KnownHostsFile: "/tmp/elyro-known-hosts",
	}
	if err := UpsertManagedSSHHost(path, entry); err != nil {
		t.Fatalf("UpsertManagedSSHHost returned error: %v", err)
	}
	if err := UpsertManagedSSHHost(path, entry); err != nil {
		t.Fatalf("UpsertManagedSSHHost second pass returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, managedSSHBegin(entry.HostAlias)) != 1 {
		t.Fatalf("managed block was not deduplicated:\n%s", content)
	}
	if !strings.Contains(content, "Host existing") {
		t.Fatalf("existing config was removed unexpectedly:\n%s", content)
	}
	for _, expected := range []string{
		`  IdentityFile "/tmp/elyro-workspace-test"`,
		"  IdentitiesOnly yes",
		"  PreferredAuthentications publickey",
		"  PasswordAuthentication no",
		"  PubkeyAuthentication yes",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected SSH config to contain %q:\n%s", expected, content)
		}
	}
	for _, insecureOption := range []string{
		"PreferredAuthentications keyboard-interactive,password",
		"PubkeyAuthentication no",
	} {
		if strings.Contains(content, insecureOption) {
			t.Fatalf("SSH config contains insecure password auth option %q:\n%s", insecureOption, content)
		}
	}
}

func TestUpsertManagedSSHHostQuotesIdentityFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	entry := SSHHostEntry{
		HostAlias:      "elyro-demo",
		HostName:       "127.0.0.1",
		Port:           "2200",
		User:           "elyro",
		IdentityFile:   filepath.Join(dir, `identity dir`, `elyro "workspace".ed25519`),
		KnownHostsFile: filepath.Join(dir, "known hosts"),
	}

	if err := UpsertManagedSSHHost(path, entry); err != nil {
		t.Fatalf("UpsertManagedSSHHost returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	expected := `  IdentityFile "` + strings.ReplaceAll(strings.ReplaceAll(entry.IdentityFile, `\`, `\\`), `"`, `\"`) + `"`
	if !strings.Contains(content, expected) {
		t.Fatalf("expected quoted identity file %q:\n%s", expected, content)
	}
}

func TestUpsertManagedSSHHostConflictsOutsideManagedBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "Host elyro-demo\n  HostName 127.0.0.1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := UpsertManagedSSHHost(path, SSHHostEntry{
		HostAlias:      "elyro-demo",
		HostName:       "127.0.0.1",
		Port:           "2200",
		User:           "elyro",
		IdentityFile:   "/tmp/elyro-workspace-test",
		KnownHostsFile: "/tmp/elyro-known-hosts",
	})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestValidateManagedSSHHostAllowsMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "missing-config")
	if err := ValidateManagedSSHHost(path, "elyro-demo"); err != nil {
		t.Fatalf("ValidateManagedSSHHost returned error for missing file: %v", err)
	}
}

func TestRemoveManagedSSHHost(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := strings.Join([]string{
		"Host keep",
		"  HostName 10.0.0.1",
		managedSSHBegin("elyro-demo"),
		"Host elyro-demo",
		"  HostName 127.0.0.1",
		managedSSHEnd("elyro-demo"),
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveManagedSSHHost(path, "elyro-demo"); err != nil {
		t.Fatalf("RemoveManagedSSHHost returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "elyro-demo") {
		t.Fatalf("managed block still present:\n%s", string(data))
	}
	if !strings.Contains(string(data), "Host keep") {
		t.Fatalf("non-managed entry was removed unexpectedly:\n%s", string(data))
	}
}

package access

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cofy-x/elyro/internal/workspace"
	"golang.org/x/crypto/ssh"
)

const DefaultWorkspaceIdentityFile = "~/.ssh/elyro_workspace_ed25519"

func EnsureSSHIdentity(path string) (string, string, error) {
	expanded, err := workspace.ExpandPath(path)
	if err != nil {
		return "", "", err
	}
	identityFile, err := filepath.Abs(expanded)
	if err != nil {
		return "", "", fmt.Errorf("resolve identity file path: %w", err)
	}
	publicKeyFile := identityFile + ".pub"

	if _, err := os.Stat(identityFile); err != nil {
		if !os.IsNotExist(err) {
			return "", "", err
		}
		if err := os.MkdirAll(filepath.Dir(identityFile), 0o700); err != nil {
			return "", "", fmt.Errorf("create identity dir: %w", err)
		}
		if err := generateSSHIdentity(identityFile, publicKeyFile); err != nil {
			return "", "", err
		}
	}
	if err := os.Chmod(identityFile, 0o600); err != nil {
		return "", "", fmt.Errorf("chmod identity file: %w", err)
	}

	publicKey, err := deriveAuthorizedKey(identityFile)
	if err != nil {
		return "", "", fmt.Errorf("derive workspace public key: %w", err)
	}
	if err := os.WriteFile(publicKeyFile, []byte(publicKey), 0o644); err != nil {
		return "", "", fmt.Errorf("write workspace public key: %w", err)
	}

	trimmedPublicKey := strings.TrimSpace(publicKey)
	if trimmedPublicKey == "" {
		return "", "", fmt.Errorf("public key %s is empty", publicKeyFile)
	}
	return identityFile, trimmedPublicKey, nil
}

func generateSSHIdentity(privateKeyFile, publicKeyFile string) error {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate workspace ssh identity: %w", err)
	}
	privateBlock, err := ssh.MarshalPrivateKey(privateKey, "elyro-workspace")
	if err != nil {
		return fmt.Errorf("marshal workspace ssh identity: %w", err)
	}
	if err := os.WriteFile(privateKeyFile, pem.EncodeToMemory(privateBlock), 0o600); err != nil {
		return fmt.Errorf("write workspace ssh identity: %w", err)
	}
	authorizedKey, err := authorizedKeyFromPrivateKey(privateKey)
	if err != nil {
		return err
	}
	if err := os.WriteFile(publicKeyFile, []byte(authorizedKey), 0o644); err != nil {
		return fmt.Errorf("write workspace public key: %w", err)
	}
	return nil
}

func deriveAuthorizedKey(privateKeyFile string) (string, error) {
	privateKey, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return "", err
	}
	rawKey, err := ssh.ParseRawPrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	return authorizedKeyFromPrivateKey(rawKey)
}

func authorizedKeyFromPrivateKey(privateKey any) (string, error) {
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(signer.PublicKey())), nil
}

func InstallContainerSSHAccess(ctx context.Context, containerName, publicKey string) error {
	script := strings.TrimSpace(`
set -eu

user_home="/home/elyro"
ssh_dir="${user_home}/.ssh"
authorized_keys="${ssh_dir}/authorized_keys"
elyro_gid="$(id -g elyro)"
begin="# ELYRO_WORKSPACE_AUTHORIZED_KEY_BEGIN"
end="# ELYRO_WORKSPACE_AUTHORIZED_KEY_END"

install -d -m 700 -o elyro -g "${elyro_gid}" "${ssh_dir}"
touch "${authorized_keys}"
chown "elyro:${elyro_gid}" "${authorized_keys}"
chmod 600 "${authorized_keys}"

tmp="$(mktemp)"
awk -v begin="${begin}" -v end="${end}" '
  $0 == begin { skip = 1; next }
  $0 == end { skip = 0; next }
  !skip { print }
' "${authorized_keys}" > "${tmp}"

if [ -s "${tmp}" ]; then
  printf '\n' >> "${tmp}"
fi
{
  printf '%s\n' "${begin}"
  printf '%s\n' "${ELYRO_WORKSPACE_PUBLIC_KEY}"
  printf '%s\n' "${end}"
} >> "${tmp}"

install -m 600 -o elyro -g "${elyro_gid}" "${tmp}" "${authorized_keys}"
rm -f "${tmp}"

install -d -m 755 /etc/ssh/sshd_config.d
cat >/etc/ssh/sshd_config.d/99-elyro-workspace.conf <<'EOF_SSHD'
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF_SSHD

sshd -t
if pgrep -x sshd >/dev/null 2>&1; then
  pkill -HUP sshd || true
fi
`)
	if _, err := runOutput(ctx, nil, "docker", "exec", "--user", "0", "-e", "ELYRO_WORKSPACE_PUBLIC_KEY="+publicKey, containerName, "bash", "-lc", script); err != nil {
		return fmt.Errorf("install workspace ssh access: %w", err)
	}
	return nil
}

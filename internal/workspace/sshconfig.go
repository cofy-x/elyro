package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SSHHostEntry struct {
	HostAlias      string
	HostName       string
	Port           string
	User           string
	IdentityFile   string
	KnownHostsFile string
}

func ValidateManagedSSHHost(path, alias string) error {
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("ssh host alias must not be empty")
	}
	content, err := readSSHConfig(path)
	if err != nil {
		return err
	}
	if hasHostAliasOutsideManagedBlock(content, alias, alias) {
		return fmt.Errorf("Host %s already exists in %s outside the managed Elyro Workspace block", alias, path)
	}
	return nil
}

func UpsertManagedSSHHost(path string, entry SSHHostEntry) error {
	if strings.TrimSpace(entry.HostAlias) == "" {
		return fmt.Errorf("ssh host alias must not be empty")
	}
	if strings.TrimSpace(entry.HostName) == "" || strings.TrimSpace(entry.Port) == "" || strings.TrimSpace(entry.User) == "" || strings.TrimSpace(entry.IdentityFile) == "" || strings.TrimSpace(entry.KnownHostsFile) == "" {
		return fmt.Errorf("ssh host entry must include hostname, port, user, identity file, and known-hosts file")
	}

	if err := ValidateManagedSSHHost(path, entry.HostAlias); err != nil {
		return err
	}

	content, err := readSSHConfig(path)
	if err != nil {
		return err
	}
	updated := removeManagedBlock(content, entry.HostAlias)
	block := buildManagedBlock(entry)
	if strings.TrimSpace(updated) == "" {
		updated = block
	} else {
		updated = strings.TrimRight(updated, "\n") + "\n\n" + block
	}
	return writeFileWithParents(path, updated)
}

func RemoveManagedSSHHost(path, alias string) error {
	content, err := readSSHConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updated := removeManagedBlock(content, alias)
	return writeFileWithParents(path, strings.TrimLeft(updated, "\n"))
}

func readSSHConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func writeFileWithParents(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dirs: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func buildManagedBlock(entry SSHHostEntry) string {
	begin := managedSSHBegin(entry.HostAlias)
	end := managedSSHEnd(entry.HostAlias)
	lines := []string{
		begin,
		fmt.Sprintf("Host %s", entry.HostAlias),
		fmt.Sprintf("  HostName %s", entry.HostName),
		fmt.Sprintf("  Port %s", entry.Port),
		fmt.Sprintf("  User %s", entry.User),
		fmt.Sprintf("  IdentityFile %s", quoteSSHConfigValue(entry.IdentityFile)),
		"  IdentitiesOnly yes",
		"  PreferredAuthentications publickey",
		"  PasswordAuthentication no",
		"  PubkeyAuthentication yes",
		"  StrictHostKeyChecking yes",
		fmt.Sprintf("  UserKnownHostsFile %s", quoteSSHConfigValue(entry.KnownHostsFile)),
		end,
	}
	return strings.Join(lines, "\n") + "\n"
}

func quoteSSHConfigValue(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func managedSSHBegin(alias string) string {
	return "# ELYRO_WORKSPACE_SSH_BEGIN " + alias
}

func managedSSHEnd(alias string) string {
	return "# ELYRO_WORKSPACE_SSH_END " + alias
}

func removeManagedBlock(content, alias string) string {
	begin := managedSSHBegin(alias)
	end := managedSSHEnd(alias)
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		switch line {
		case begin:
			inBlock = true
			continue
		case end:
			inBlock = false
			continue
		}
		if !inBlock {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimRight(strings.Join(filtered, "\n"), "\n") + "\n"
}

func hasHostAliasOutsideManagedBlock(content, alias, managedAlias string) bool {
	begin := managedSSHBegin(managedAlias)
	end := managedSSHEnd(managedAlias)
	lines := strings.Split(content, "\n")
	inBlock := false
	for _, line := range lines {
		switch line {
		case begin:
			inBlock = true
			continue
		case end:
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Host") {
			for _, candidate := range fields[1:] {
				if candidate == alias {
					return true
				}
			}
		}
	}
	return false
}

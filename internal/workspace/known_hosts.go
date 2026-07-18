package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func DefaultKnownHostsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".ssh", "elyro_known_hosts")
	}
	return filepath.Join(home, ".ssh", "elyro_known_hosts")
}

func PrepareKnownSSHHost(ctx context.Context, path, alias, containerID, host, port string) error {
	return prepareKnownSSHHost(ctx, path, alias, containerID, host, port, scanSSHHostKeys)
}

func prepareKnownSSHHost(ctx context.Context, path, alias, containerID, host, port string, scan func(context.Context, string, string) (string, error)) error {
	keys, err := scan(ctx, host, port)
	if err != nil {
		return err
	}
	content, err := readSSHConfig(path)
	if err != nil {
		return err
	}
	previousID, previousKeys := knownHostBlock(content, alias)
	if previousID == containerID && previousKeys != "" && previousKeys != keys {
		return fmt.Errorf("SSH host key changed unexpectedly for running workspace %s; remove the container only if this change is expected", alias)
	}
	updated := removeKnownHostBlock(content, alias)
	block := buildKnownHostBlock(alias, containerID, keys)
	if strings.TrimSpace(updated) == "" {
		updated = block
	} else {
		updated = strings.TrimRight(updated, "\n") + "\n\n" + block
	}
	return writeFileWithParents(path, updated)
}

func RemoveKnownSSHHost(path, alias string) error {
	content, err := readSSHConfig(path)
	if err != nil {
		return err
	}
	return writeFileWithParents(path, strings.TrimLeft(removeKnownHostBlock(content, alias), "\n"))
}

func scanSSHHostKeys(ctx context.Context, host, port string) (string, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		keys, err := scanSSHHostKeysOnce(ctx, host, port)
		if err == nil || !isTransientSSHScanError(err) || time.Now().After(deadline) {
			return keys, err
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

func scanSSHHostKeysOnce(ctx context.Context, host, port string) (string, error) {
	cmd := exec.CommandContext(ctx, "ssh-keyscan", "-T", "10", "-p", port, host)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("scan workspace SSH host key: %w: %s", err, message)
		}
		return "", fmt.Errorf("scan workspace SSH host key: %w", err)
	}
	var lines []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return "", errorsNewNoSSHKeys(host, port)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func isTransientSSHScanError(err error) bool {
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"connection closed",
		"connection refused",
		"connection reset",
		"operation timed out",
		"i/o timeout",
		"no keys returned",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func errorsNewNoSSHKeys(host, port string) error {
	return fmt.Errorf("scan workspace SSH host key: no keys returned for %s:%s", host, port)
}

func knownHostBegin(alias, containerID string) string {
	return fmt.Sprintf("# ELYRO_WORKSPACE_KNOWN_HOST_BEGIN %s %s", alias, containerID)
}

func knownHostEnd(alias string) string { return "# ELYRO_WORKSPACE_KNOWN_HOST_END " + alias }

func buildKnownHostBlock(alias, containerID, keys string) string {
	return knownHostBegin(alias, containerID) + "\n" + keys + "\n" + knownHostEnd(alias) + "\n"
}

func knownHostBlock(content, alias string) (string, string) {
	var id string
	var keys []string
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		prefix := "# ELYRO_WORKSPACE_KNOWN_HOST_BEGIN " + alias + " "
		if strings.HasPrefix(line, prefix) {
			id = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			inBlock = true
			continue
		}
		if line == knownHostEnd(alias) {
			inBlock = false
			continue
		}
		if inBlock && strings.TrimSpace(line) != "" {
			keys = append(keys, strings.TrimSpace(line))
		}
	}
	return id, strings.Join(keys, "\n")
}

func removeKnownHostBlock(content, alias string) string {
	var out []string
	inBlock := false
	prefix := "# ELYRO_WORKSPACE_KNOWN_HOST_BEGIN " + alias + " "
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			inBlock = true
			continue
		}
		if inBlock && line == knownHostEnd(alias) {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

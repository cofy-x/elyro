package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/spf13/cobra"
)

func newShellCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Open a Linux shell",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			record, err := currentWorkspace(cmd, opts)
			if err != nil {
				return err
			}
			ctx, cancel := signalContext()
			defer cancel()
			args := dockerShellArgs(record)
			if err := runStreamingIO(ctx, "", cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "docker", args...); err != nil {
				return fmt.Errorf("open shell in workspace %s: %w; run `elyro status` to inspect it", record.Name, err)
			}
			return nil
		},
	}
}

func newExecCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "exec -- COMMAND [ARG...]",
		Short: "Run a command in Linux",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, command []string) error {
			record, err := currentWorkspace(cmd, opts)
			if err != nil {
				return err
			}
			ctx, cancel := signalContext()
			defer cancel()
			pidFile, err := newExecPIDFile()
			if err != nil {
				return fmt.Errorf("prepare workspace command tracking: %w", err)
			}
			args := dockerExecArgs(record, pidFile, command)
			runErr := runStreamingIO(ctx, "", cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "docker", args...)
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cleanupCancel()
			if cleanupErr := cleanupDockerExec(cleanupCtx, record.ContainerName, pidFile, ctx.Err() != nil); cleanupErr != nil {
				return fmt.Errorf("clean up command in workspace %s: %w", record.Name, cleanupErr)
			}
			if runErr != nil {
				return fmt.Errorf("execute command in workspace %s: %w", record.Name, runErr)
			}
			return nil
		},
	}
}

func dockerShellArgs(record elyroworkspace.Record) []string {
	args := []string{"exec", "-it"}
	if noColor := os.Getenv("NO_COLOR"); noColor != "" {
		args = append(args, "--env", "NO_COLOR="+noColor)
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		args = append(args, "--env", "TERM=dumb")
	}
	return append(args,
		"--user", "elyro", "--workdir", record.ContainerWorkspaceDir,
		record.ContainerName, "/bin/sh", "-c",
		`shell="$(getent passwd elyro 2>/dev/null | awk -F: '{print $7}')"; [ -x "$shell" ] || shell=/bin/bash; exec "$shell" -l`,
	)
}

func dockerExecArgs(record elyroworkspace.Record, pidFile string, command []string) []string {
	args := []string{
		"exec", "-i", "--user", "elyro", "--workdir", record.ContainerWorkspaceDir,
		record.ContainerName, "/bin/sh", "-c",
		`pid_file="$1"; shift; umask 077; printf '%s\n' "$$" >"$pid_file"; exec "$@"`,
		"elyro-exec", pidFile,
	}
	return append(args, command...)
}

func newExecPIDFile() (string, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	return "/tmp/elyro-exec-" + hex.EncodeToString(token[:]) + ".pid", nil
}

func cleanupDockerExec(ctx context.Context, containerName, pidFile string, interrupted bool) error {
	mode := "remove"
	if interrupted {
		mode = "interrupt"
	}
	script := `
pid_file="$1"
mode="$2"
if [ ! -f "$pid_file" ]; then exit 0; fi
pid="$(cat "$pid_file")"
case "$pid" in ''|*[!0-9]*) rm -f "$pid_file"; exit 1 ;; esac
if [ "$mode" = interrupt ]; then
  /bin/kill -INT -- "-$pid" 2>/dev/null || true
  count=0
  while /bin/kill -0 -- "-$pid" 2>/dev/null && [ "$count" -lt 50 ]; do
    sleep 0.1
    count=$((count + 1))
  done
  /bin/kill -KILL -- "-$pid" 2>/dev/null || true
fi
rm -f "$pid_file"
`
	return runStreamingIO(ctx, "", nil, nil, nil, "docker", "exec", "--user", "0", containerName, "/bin/sh", "-c", script, "elyro-exec-cleanup", pidFile, mode)
}

func currentWorkspace(cmd *cobra.Command, opts *GlobalOptions) (elyroworkspace.Record, error) {
	projectDir, err := resolvedProjectDir(cmd, opts)
	if err != nil {
		return elyroworkspace.Record{}, err
	}
	store, _, err := loadWorkspaceStore()
	if err != nil {
		return elyroworkspace.Record{}, err
	}
	record, err := elyroworkspace.Current(store, projectDir)
	if err != nil {
		if errors.Is(err, elyroworkspace.ErrNoCurrent) {
			return elyroworkspace.Record{}, errors.New("no current workspace found; run `elyro up` from this project first")
		}
		return elyroworkspace.Record{}, err
	}
	return record, nil
}

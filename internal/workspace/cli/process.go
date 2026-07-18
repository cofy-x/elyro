package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"
)

func runStreamingIO(ctx context.Context, dir string, in io.Reader, out, errOut io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
		return nil
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Stdin = in
	return cmd.Run()
}

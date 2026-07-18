package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunStreamingIOInterruptsBeforeForcedCancellation(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	interrupted := filepath.Join(dir, "interrupted")
	script := filepath.Join(dir, "wait.sh")
	contents := "#!/bin/sh\n" +
		"trap 'touch \"$2\"; exit 0' INT\n" +
		"touch \"$1\"\n" +
		"while :; do sleep 0.05; done\n"
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runStreamingIO(ctx, dir, nil, io.Discard, io.Discard, script, ready, interrupted)
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("command did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runStreamingIO() error = %v, want context.Canceled", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("runStreamingIO() did not stop after cancellation")
	}
	if _, err := os.Stat(interrupted); err != nil {
		t.Fatalf("command did not receive interrupt: %v", err)
	}
}

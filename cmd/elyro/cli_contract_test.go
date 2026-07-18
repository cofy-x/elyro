package main

import (
	"sort"
	"testing"
)

func TestTopLevelCLIContract(t *testing.T) {
	want := []string{"doctor", "down", "exec", "init", "list", "open", "shell", "skill", "status", "up", "version"}
	cmd := newRootCmd()
	got := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Name() != "help" {
			got = append(got, child.Name())
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("top-level commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("top-level commands = %v, want %v", got, want)
		}
	}
}

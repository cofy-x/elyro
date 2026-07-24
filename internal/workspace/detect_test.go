package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectToolchains(t *testing.T) {
	tests := []struct {
		name    string
		markers []string
		want    []Toolchain
	}{
		{name: "python", markers: []string{"pyproject.toml", "uv.lock"}, want: []Toolchain{ToolchainPython}},
		{name: "go", markers: []string{"go.mod"}, want: []Toolchain{ToolchainGo}},
		{name: "node", markers: []string{"package.json", "pnpm-lock.yaml"}, want: []Toolchain{ToolchainNode}},
		{name: "multiple", markers: []string{"requirements.txt", "go.work"}, want: []Toolchain{ToolchainPython, ToolchainGo}},
		{name: "none", want: []Toolchain{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, marker := range tt.markers {
				if err := os.WriteFile(filepath.Join(dir, marker), []byte("marker\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := DetectToolchains(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("DetectToolchains() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDetectToolchainRequiresUniqueMatch(t *testing.T) {
	dir := t.TempDir()
	_, err := DetectToolchain(dir)
	var detectionErr *ToolchainDetectionError
	if !errors.As(err, &detectionErr) || len(detectionErr.Matches) != 0 {
		t.Fatalf("DetectToolchain() error = %v, want empty ToolchainDetectionError", err)
	}
}

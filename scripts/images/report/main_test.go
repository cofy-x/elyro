package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	outputs map[string][]byte
	errors  map[string]error
}

func (f fakeRunner) Run(args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	if err := f.errors[key]; err != nil {
		return nil, err
	}
	value, ok := f.outputs[key]
	if !ok {
		return nil, errors.New("unexpected docker command: " + key)
	}
	return value, nil
}

func TestInspectVersionCollectsPlatformsAndHistory(t *testing.T) {
	value, err := inspectVersion(fixtureRunner(true), "registry.example/elyro", "workspace-base", "v1.2.3", true)
	if err != nil {
		t.Fatal(err)
	}
	if value.IndexDigest != "sha256:index123" {
		t.Fatalf("IndexDigest = %q", value.IndexDigest)
	}
	if value.Platforms["amd64"].Bytes != 300 || value.Platforms["arm64"].Bytes != 270 {
		t.Fatalf("platform bytes = amd64:%d arm64:%d", value.Platforms["amd64"].Bytes, value.Platforms["arm64"].Bytes)
	}
	if got := value.Platforms["amd64"].Layers[1].CreatedBy; got != "RUN install tool" {
		t.Fatalf("second layer CreatedBy = %q", got)
	}
}

func TestBuildImageReportCalculatesDeltaAndJSON(t *testing.T) {
	target, err := inspectVersion(fixtureRunner(false), "registry.example/elyro", "workspace-base", "v1.2.3", false)
	if err != nil {
		t.Fatal(err)
	}
	comparison := target
	comparison.Platforms = map[string]platformImage{
		"amd64": {Bytes: 200},
		"arm64": {Bytes: 300},
	}
	value := report{ImagePrefix: "registry.example/elyro", Version: "v1.2.3", CompareVersion: "v1.2.2", Images: []imageReport{buildImageReport("workspace-base", target, &comparison, 1)}}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	if err := encoder.Encode(value); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{`"delta_bytes":100`, `"delta_percent":50`, `"top_layers"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("JSON missing %s: %s", want, text)
		}
	}
}

func TestInspectVersionRequiresBothArchitectures(t *testing.T) {
	runner := fixtureRunner(false)
	key := "buildx imagetools inspect --raw registry.example/elyro/workspace-base:v1.2.3"
	runner.outputs[key] = []byte(`{"manifests":[{"digest":"sha256:amd","platform":{"os":"linux","architecture":"amd64"}}]}`)
	_, err := inspectVersion(runner, "registry.example/elyro", "workspace-base", "v1.2.3", false)
	if err == nil || !strings.Contains(err.Error(), "missing required platform linux/arm64") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateConfigRejectsFloatingAndInvalidInputs(t *testing.T) {
	for _, cfg := range []config{
		{Version: "latest", Format: "table", ImagePrefix: "x"},
		{Version: "0.3.0", Format: "table", ImagePrefix: "x"},
		{Version: "v0.3.0", CompareVersion: "stable", Format: "table", ImagePrefix: "x"},
		{Version: "v0.3.0", Format: "yaml", ImagePrefix: "x"},
		{Version: "v0.3.0", Format: "table", ImagePrefix: "x", TopLayers: -1},
	} {
		if err := validateConfig(cfg); err == nil {
			t.Fatalf("validateConfig(%+v) succeeded", cfg)
		}
	}
}

func TestValidateConfigAllowsOnlyImmutableCandidateTagsWhenEnabled(t *testing.T) {
	cfg := config{
		Version:          "candidate-0123456789abcdef0123456789abcdef01234567-4242",
		Format:           "json",
		ImagePrefix:      "registry.example/elyro",
		AllowCandidate:   true,
		ExpectedRevision: "0123456789abcdef0123456789abcdef01234567",
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Version = "candidate-main-4242"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("validateConfig accepted a floating candidate tag")
	}
}

func TestValidateCandidateReportChecksLabelsAndBudgets(t *testing.T) {
	revision := "0123456789abcdef0123456789abcdef01234567"
	version := "candidate-" + revision + "-4242"
	labels := map[string]string{
		"org.opencontainers.image.revision": revision,
		"org.opencontainers.image.version":  version,
		"org.opencontainers.image.source":   "https://github.com/cofy-x/elyro",
		"org.opencontainers.image.url":      "https://github.com/cofy-x/elyro",
		"org.opencontainers.image.licenses": "Apache-2.0",
	}
	value := report{Version: version, Images: []imageReport{{
		Name: "workspace-base",
		Platforms: []platformReport{
			{Architecture: "amd64", CompressedMiB: 70, Labels: labels},
			{Architecture: "arm64", CompressedMiB: 69, Labels: labels},
		},
	}}}
	budgetFile := filepath.Join(t.TempDir(), "budgets.json")
	if err := os.WriteFile(budgetFile, []byte(`{"schema":1,"unit":"MiB","images":{"workspace-base":90}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config{Version: version, ExpectedRevision: revision, ExpectedSource: "https://github.com/cofy-x/elyro", BudgetFile: budgetFile}
	if err := validateCandidateReport(value, cfg); err != nil {
		t.Fatal(err)
	}
	value.Images[0].Platforms[0].CompressedMiB = 91
	if err := validateCandidateReport(value, cfg); err == nil || !strings.Contains(err.Error(), "exceeding") {
		t.Fatalf("budget error = %v", err)
	}
	value.Images[0].Platforms[0].CompressedMiB = 70
	value.Images[0].Platforms[0].Labels["org.opencontainers.image.revision"] = strings.Repeat("f", 40)
	if err := validateCandidateReport(value, cfg); err == nil || !strings.Contains(err.Error(), "revision label") {
		t.Fatalf("label error = %v", err)
	}
}

func TestDiscoverImagesAndReleaseMatrixMustMatch(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"workspace-base", "workspace-go"} {
		dir := filepath.Join(root, "images", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	images, err := discoverImages(filepath.Join(root, "images"))
	if err != nil {
		t.Fatal(err)
	}
	releaseFile := filepath.Join(root, "release.yml")
	if err := os.WriteFile(releaseFile, []byte("matrix:\n  include:\n    - image: workspace-go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseMatrix(images, releaseFile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(releaseFile, []byte("matrix:\n  include:\n    - image: other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseMatrix(images, releaseFile); err == nil {
		t.Fatal("validateReleaseMatrix succeeded for drifted matrix")
	}
}

func fixtureRunner(includeMetadata bool) fakeRunner {
	ref := "registry.example/elyro/workspace-base:v1.2.3"
	outputs := map[string][]byte{
		"buildx version": []byte("buildx v1"),
		"buildx imagetools inspect " + ref + " --format {{json .Manifest.Digest}}":         []byte(`"sha256:index123"`),
		"buildx imagetools inspect --raw " + ref:                                           []byte(`{"manifests":[{"digest":"sha256:amd","platform":{"os":"linux","architecture":"amd64"}},{"digest":"sha256:arm","platform":{"os":"linux","architecture":"arm64"}},{"digest":"sha256:attestation","platform":{"os":"unknown","architecture":"unknown"}}]}`),
		"buildx imagetools inspect --raw registry.example/elyro/workspace-base@sha256:amd": []byte(`{"layers":[{"digest":"sha256:a1","size":100},{"digest":"sha256:a2","size":200}]}`),
		"buildx imagetools inspect --raw registry.example/elyro/workspace-base@sha256:arm": []byte(`{"layers":[{"digest":"sha256:b1","size":120},{"digest":"sha256:b2","size":150}]}`),
	}
	if includeMetadata {
		outputs["buildx imagetools inspect "+ref+" --format {{json .Image}}"] = []byte(`{"linux/amd64":{"history":[{"created_by":"metadata","empty_layer":true},{"created_by":"ADD base"},{"created_by":"RUN install tool"}]},"linux/arm64":{"history":[{"created_by":"ADD base"},{"created_by":"RUN install tool"}]}}`)
	}
	return fakeRunner{outputs: outputs, errors: map[string]error{}}
}

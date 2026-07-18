package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var exactVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)
var candidateTagPattern = regexp.MustCompile(`^candidate-[0-9a-f]{40}-[0-9]+$`)
var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var releaseImagePattern = regexp.MustCompile(`(?m)^\s*- image:\s*([a-z0-9][a-z0-9-]*)\s*$`)

type config struct {
	Version          string
	CompareVersion   string
	Format           string
	ImagePrefix      string
	ImagesDir        string
	ReleaseFile      string
	TopLayers        int
	AllowCandidate   bool
	ExpectedRevision string
	ExpectedSource   string
	BudgetFile       string
}

type commandRunner interface {
	Run(args ...string) ([]byte, error)
}

type dockerRunner struct{}

func (dockerRunner) Run(args ...string) ([]byte, error) {
	var lastError error
	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "docker", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		cancel()
		if err == nil {
			return output, nil
		}
		message := strings.TrimSpace(stderr.String())
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			message = "request timed out after 30s"
		}
		if message == "" {
			message = err.Error()
		}
		lastError = errors.New(message)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, lastError
}

type platformDescriptor struct {
	Digest   string `json:"digest"`
	Platform struct {
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
	} `json:"platform"`
}

type imageIndex struct {
	Manifests []platformDescriptor `json:"manifests"`
}

type imageManifest struct {
	Layers []layerDescriptor `json:"layers"`
}

type layerDescriptor struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type imageMetadata struct {
	History []historyEntry `json:"history"`
	Config  struct {
		Labels map[string]string `json:"Labels"`
	} `json:"config"`
}

type historyEntry struct {
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer"`
}

type report struct {
	ImagePrefix    string        `json:"image_prefix"`
	Version        string        `json:"version"`
	CompareVersion string        `json:"compare_version,omitempty"`
	Images         []imageReport `json:"images"`
}

type imageReport struct {
	Name        string           `json:"name"`
	IndexDigest string           `json:"index_digest"`
	Platforms   []platformReport `json:"platforms"`
}

type platformReport struct {
	OS                     string            `json:"os"`
	Architecture           string            `json:"architecture"`
	ManifestDigest         string            `json:"manifest_digest"`
	CompressedBytes        int64             `json:"compressed_bytes"`
	CompressedMiB          float64           `json:"compressed_mib"`
	CompareCompressedBytes *int64            `json:"compare_compressed_bytes,omitempty"`
	DeltaBytes             *int64            `json:"delta_bytes,omitempty"`
	DeltaPercent           *float64          `json:"delta_percent,omitempty"`
	TopLayers              []layerReport     `json:"top_layers,omitempty"`
	Labels                 map[string]string `json:"labels,omitempty"`
}

type layerReport struct {
	Digest          string  `json:"digest"`
	CompressedBytes int64   `json:"compressed_bytes"`
	CompressedMiB   float64 `json:"compressed_mib"`
	CreatedBy       string  `json:"created_by,omitempty"`
}

type versionImage struct {
	IndexDigest string
	Platforms   map[string]platformImage
}

type platformImage struct {
	Digest string
	Layers []layerReport
	Bytes  int64
	Labels map[string]string
}

type imageBudgets struct {
	Schema int                `json:"schema"`
	Unit   string             `json:"unit"`
	Images map[string]float64 `json:"images"`
}

func main() {
	cfg := parseFlags()
	if err := run(cfg, dockerRunner{}, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "image-report: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.Version, "version", "", "exact v-prefixed image version")
	flag.StringVar(&cfg.CompareVersion, "compare-version", "", "optional exact comparison version")
	flag.StringVar(&cfg.Format, "format", "table", "output format: table or json")
	flag.StringVar(&cfg.ImagePrefix, "image-prefix", "ghcr.io/cofy-x/elyro", "image registry prefix")
	flag.StringVar(&cfg.ImagesDir, "images-dir", "images", "supported image definitions directory")
	flag.StringVar(&cfg.ReleaseFile, "release-file", ".github/workflows/release.yml", "release workflow to validate")
	flag.IntVar(&cfg.TopLayers, "top-layers", 0, "include N largest layers for the target version")
	flag.BoolVar(&cfg.AllowCandidate, "allow-candidate", false, "allow an immutable candidate-<sha>-<run-id> tag")
	flag.StringVar(&cfg.ExpectedRevision, "expected-revision", "", "require OCI revision and release labels on every platform")
	flag.StringVar(&cfg.ExpectedSource, "expected-source", "", "require exact OCI source and URL labels on every platform")
	flag.StringVar(&cfg.BudgetFile, "budget-file", "", "optional JSON file containing per-image compressed MiB budgets")
	flag.Parse()
	return cfg
}

func run(cfg config, runner commandRunner, output io.Writer) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("required command not found: docker")
	}
	if _, err := runner.Run("buildx", "version"); err != nil {
		return fmt.Errorf("Docker Buildx is required: %w", err)
	}
	images, err := discoverImages(cfg.ImagesDir)
	if err != nil {
		return err
	}
	if err := validateReleaseMatrix(images, cfg.ReleaseFile); err != nil {
		return err
	}

	result := report{ImagePrefix: strings.TrimRight(cfg.ImagePrefix, "/"), Version: cfg.Version, CompareVersion: cfg.CompareVersion, Images: make([]imageReport, len(images))}
	semaphore := make(chan struct{}, 3)
	errorsByImage := make([]error, len(images))
	var wait sync.WaitGroup
	for index, name := range images {
		wait.Add(1)
		go func(index int, name string) {
			defer wait.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			target, inspectErr := inspectVersion(runner, result.ImagePrefix, name, cfg.Version, cfg.TopLayers > 0 || cfg.ExpectedRevision != "")
			if inspectErr != nil {
				errorsByImage[index] = fmt.Errorf("%s:%s: %w", name, cfg.Version, inspectErr)
				return
			}
			var comparison *versionImage
			if cfg.CompareVersion != "" {
				value, compareErr := inspectVersion(runner, result.ImagePrefix, name, cfg.CompareVersion, false)
				if compareErr != nil {
					errorsByImage[index] = fmt.Errorf("%s:%s: %w", name, cfg.CompareVersion, compareErr)
					return
				}
				comparison = &value
			}
			result.Images[index] = buildImageReport(name, target, comparison, cfg.TopLayers)
		}(index, name)
	}
	wait.Wait()
	for _, inspectErr := range errorsByImage {
		if inspectErr != nil {
			return inspectErr
		}
	}
	if err := validateCandidateReport(result, cfg); err != nil {
		return err
	}
	if cfg.Format == "json" {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	writeTable(output, result)
	return nil
}

func validateConfig(cfg config) error {
	if !exactVersionPattern.MatchString(cfg.Version) && !(cfg.AllowCandidate && candidateTagPattern.MatchString(cfg.Version)) {
		return fmt.Errorf("VERSION must be an exact v-prefixed version, got %q", cfg.Version)
	}
	if cfg.CompareVersion != "" && !exactVersionPattern.MatchString(cfg.CompareVersion) {
		return fmt.Errorf("COMPARE_VERSION must be an exact v-prefixed version, got %q", cfg.CompareVersion)
	}
	if cfg.Format != "table" && cfg.Format != "json" {
		return fmt.Errorf("FORMAT must be table or json, got %q", cfg.Format)
	}
	if strings.TrimSpace(cfg.ImagePrefix) == "" {
		return errors.New("ELYRO_IMAGE_PREFIX must not be empty")
	}
	if cfg.TopLayers < 0 {
		return errors.New("TOP_LAYERS must be >= 0")
	}
	if cfg.ExpectedRevision != "" && !revisionPattern.MatchString(cfg.ExpectedRevision) {
		return fmt.Errorf("EXPECTED_REVISION must be a lowercase 40-character Git SHA, got %q", cfg.ExpectedRevision)
	}
	if cfg.ExpectedRevision != "" && !candidateTagPattern.MatchString(cfg.Version) && !exactVersionPattern.MatchString(cfg.Version) {
		return errors.New("EXPECTED_REVISION requires an exact release or immutable candidate tag")
	}
	if cfg.ExpectedSource != "" && cfg.ExpectedRevision == "" {
		return errors.New("EXPECTED_SOURCE requires EXPECTED_REVISION")
	}
	return nil
}

func discoverImages(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*", "Dockerfile"))
	if err != nil {
		return nil, fmt.Errorf("discover supported images: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no supported image Dockerfiles found under %s", dir)
	}
	images := make([]string, 0, len(matches))
	for _, match := range matches {
		images = append(images, filepath.Base(filepath.Dir(match)))
	}
	sort.Strings(images)
	return images, nil
}

func validateReleaseMatrix(images []string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read release workflow: %w", err)
	}
	matches := releaseImagePattern.FindAllSubmatch(content, -1)
	releaseImages := make([]string, 0, len(matches)+1)
	releaseImages = append(releaseImages, "workspace-base")
	for _, match := range matches {
		releaseImages = append(releaseImages, string(match[1]))
	}
	sort.Strings(releaseImages)
	if strings.Join(images, "\n") != strings.Join(releaseImages, "\n") {
		return fmt.Errorf("supported image definitions do not match release matrix: definitions=%s release=%s", strings.Join(images, ","), strings.Join(releaseImages, ","))
	}
	return nil
}

func inspectVersion(runner commandRunner, prefix, name, version string, includeHistory bool) (versionImage, error) {
	ref := fmt.Sprintf("%s/%s:%s", prefix, name, version)
	indexDigestOutput, err := runner.Run("buildx", "imagetools", "inspect", ref, "--format", "{{json .Manifest.Digest}}")
	if err != nil {
		return versionImage{}, fmt.Errorf("inspect index digest: %w", err)
	}
	indexDigest, err := decodeFormattedString(indexDigestOutput)
	if err != nil || !strings.HasPrefix(indexDigest, "sha256:") {
		return versionImage{}, fmt.Errorf("invalid index digest %q", strings.TrimSpace(string(indexDigestOutput)))
	}
	rawIndex, err := runner.Run("buildx", "imagetools", "inspect", "--raw", ref)
	if err != nil {
		return versionImage{}, fmt.Errorf("inspect index: %w", err)
	}
	var index imageIndex
	if err := json.Unmarshal(rawIndex, &index); err != nil {
		return versionImage{}, fmt.Errorf("decode index: %w", err)
	}

	metadata := map[string]imageMetadata{}
	if includeHistory {
		rawMetadata, metadataErr := runner.Run("buildx", "imagetools", "inspect", ref, "--format", "{{json .Image}}")
		if metadataErr != nil {
			return versionImage{}, fmt.Errorf("inspect image history: %w", metadataErr)
		}
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return versionImage{}, fmt.Errorf("decode image history: %w", err)
		}
	}

	result := versionImage{IndexDigest: indexDigest, Platforms: map[string]platformImage{}}
	for _, descriptor := range index.Manifests {
		if descriptor.Platform.OS != "linux" || (descriptor.Platform.Architecture != "amd64" && descriptor.Platform.Architecture != "arm64") {
			continue
		}
		arch := descriptor.Platform.Architecture
		if _, exists := result.Platforms[arch]; exists {
			return versionImage{}, fmt.Errorf("duplicate linux/%s manifest", arch)
		}
		rawManifest, manifestErr := runner.Run("buildx", "imagetools", "inspect", "--raw", fmt.Sprintf("%s/%s@%s", prefix, name, descriptor.Digest))
		if manifestErr != nil {
			return versionImage{}, fmt.Errorf("inspect linux/%s manifest: %w", arch, manifestErr)
		}
		var manifest imageManifest
		if err := json.Unmarshal(rawManifest, &manifest); err != nil {
			return versionImage{}, fmt.Errorf("decode linux/%s manifest: %w", arch, err)
		}
		if len(manifest.Layers) == 0 {
			return versionImage{}, fmt.Errorf("linux/%s manifest has no layers", arch)
		}
		createdBy := nonEmptyHistory(metadata["linux/"+arch].History)
		platform := platformImage{Digest: descriptor.Digest, Labels: metadata["linux/"+arch].Config.Labels}
		for index, layer := range manifest.Layers {
			if layer.Size < 0 || layer.Digest == "" {
				return versionImage{}, fmt.Errorf("linux/%s manifest contains invalid layer", arch)
			}
			entry := layerReport{Digest: layer.Digest, CompressedBytes: layer.Size, CompressedMiB: mib(layer.Size)}
			if index < len(createdBy) {
				entry.CreatedBy = createdBy[index]
			}
			platform.Bytes += layer.Size
			platform.Layers = append(platform.Layers, entry)
		}
		result.Platforms[arch] = platform
	}
	for _, arch := range []string{"amd64", "arm64"} {
		if _, ok := result.Platforms[arch]; !ok {
			return versionImage{}, fmt.Errorf("missing required platform linux/%s", arch)
		}
	}
	return result, nil
}

func decodeFormattedString(value []byte) (string, error) {
	var decoded string
	if err := json.Unmarshal(bytes.TrimSpace(value), &decoded); err == nil {
		return decoded, nil
	}
	trimmed := strings.Trim(strings.TrimSpace(string(value)), `"`)
	if trimmed == "" {
		return "", errors.New("empty formatted value")
	}
	return trimmed, nil
}

func nonEmptyHistory(history []historyEntry) []string {
	result := make([]string, 0, len(history))
	for _, entry := range history {
		if !entry.EmptyLayer {
			result = append(result, entry.CreatedBy)
		}
	}
	return result
}

func buildImageReport(name string, target versionImage, comparison *versionImage, topLayers int) imageReport {
	result := imageReport{Name: name, IndexDigest: target.IndexDigest}
	for _, arch := range []string{"amd64", "arm64"} {
		value := target.Platforms[arch]
		platform := platformReport{OS: "linux", Architecture: arch, ManifestDigest: value.Digest, CompressedBytes: value.Bytes, CompressedMiB: mib(value.Bytes), Labels: value.Labels}
		if comparison != nil {
			compareBytes := comparison.Platforms[arch].Bytes
			delta := value.Bytes - compareBytes
			platform.CompareCompressedBytes = &compareBytes
			platform.DeltaBytes = &delta
			if compareBytes != 0 {
				percent := float64(delta) / float64(compareBytes) * 100
				platform.DeltaPercent = &percent
			}
		}
		layers := append([]layerReport(nil), value.Layers...)
		sort.SliceStable(layers, func(i, j int) bool { return layers[i].CompressedBytes > layers[j].CompressedBytes })
		limit := topLayers
		if limit > len(layers) {
			limit = len(layers)
		}
		if limit > 0 {
			platform.TopLayers = layers[:limit]
		}
		result.Platforms = append(result.Platforms, platform)
	}
	return result
}

func validateCandidateReport(value report, cfg config) error {
	if cfg.ExpectedRevision != "" {
		for _, image := range value.Images {
			for _, platform := range image.Platforms {
				labels := platform.Labels
				if labels["org.opencontainers.image.revision"] != cfg.ExpectedRevision {
					return fmt.Errorf("%s linux/%s revision label does not match %s", image.Name, platform.Architecture, cfg.ExpectedRevision)
				}
				if labels["org.opencontainers.image.version"] != cfg.Version {
					return fmt.Errorf("%s linux/%s version label does not match %s", image.Name, platform.Architecture, cfg.Version)
				}
				if cfg.ExpectedSource != "" && (labels["org.opencontainers.image.source"] != cfg.ExpectedSource || labels["org.opencontainers.image.url"] != cfg.ExpectedSource) {
					return fmt.Errorf("%s linux/%s OCI source labels do not match %s", image.Name, platform.Architecture, cfg.ExpectedSource)
				}
				if labels["org.opencontainers.image.source"] == "" || labels["org.opencontainers.image.url"] == "" {
					return fmt.Errorf("%s linux/%s is missing OCI source labels", image.Name, platform.Architecture)
				}
				if labels["org.opencontainers.image.licenses"] != "Apache-2.0" {
					return fmt.Errorf("%s linux/%s has unexpected OCI license label", image.Name, platform.Architecture)
				}
			}
		}
	}
	if cfg.BudgetFile == "" {
		return nil
	}
	content, err := os.ReadFile(cfg.BudgetFile)
	if err != nil {
		return fmt.Errorf("read image budgets: %w", err)
	}
	var budgets imageBudgets
	if err := json.Unmarshal(content, &budgets); err != nil {
		return fmt.Errorf("decode image budgets: %w", err)
	}
	if budgets.Schema != 1 || budgets.Unit != "MiB" {
		return fmt.Errorf("image budgets must use schema 1 and MiB units")
	}
	if len(budgets.Images) != len(value.Images) {
		return fmt.Errorf("image budget count %d does not match report count %d", len(budgets.Images), len(value.Images))
	}
	for _, image := range value.Images {
		limit, ok := budgets.Images[image.Name]
		if !ok || limit <= 0 {
			return fmt.Errorf("missing positive budget for %s", image.Name)
		}
		for _, platform := range image.Platforms {
			if platform.CompressedMiB > limit {
				return fmt.Errorf("%s linux/%s is %.3f MiB, exceeding %.3f MiB budget", image.Name, platform.Architecture, platform.CompressedMiB, limit)
			}
		}
	}
	return nil
}

func mib(bytes int64) float64 {
	return float64(bytes) / 1024 / 1024
}

func writeTable(output io.Writer, value report) {
	fmt.Fprintf(output, "Image prefix: %s\nVersion: %s\n", value.ImagePrefix, value.Version)
	if value.CompareVersion != "" {
		fmt.Fprintf(output, "Compare: %s\n", value.CompareVersion)
	}
	fmt.Fprintln(output)
	fmt.Fprintln(output, "IMAGE\tPLATFORM\tCOMPRESSED\tCOMPARE\tDELTA\tMANIFEST")
	for _, image := range value.Images {
		fmt.Fprintf(output, "%s index: %s\n", image.Name, image.IndexDigest)
		for _, platform := range image.Platforms {
			compare := "-"
			delta := "-"
			if platform.CompareCompressedBytes != nil {
				compare = fmt.Sprintf("%.1f MiB", mib(*platform.CompareCompressedBytes))
			}
			if platform.DeltaBytes != nil && platform.DeltaPercent != nil {
				delta = fmt.Sprintf("%+.1f MiB (%+.1f%%)", mib(*platform.DeltaBytes), *platform.DeltaPercent)
			}
			fmt.Fprintf(output, "%s\tlinux/%s\t%.1f MiB\t%s\t%s\t%s\n", image.Name, platform.Architecture, platform.CompressedMiB, compare, delta, platform.ManifestDigest)
			for index, layer := range platform.TopLayers {
				createdBy := strings.Join(strings.Fields(layer.CreatedBy), " ")
				fmt.Fprintf(output, "  layer %d\tlinux/%s\t%.1f MiB\t-\t-\t%s\n", index+1, platform.Architecture, layer.CompressedMiB, layer.Digest)
				if createdBy != "" {
					fmt.Fprintf(output, "    created_by: %s\n", abbreviate(createdBy, 240))
				}
			}
		}
	}
}

func abbreviate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

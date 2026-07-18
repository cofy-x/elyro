package images

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	elyroversion "github.com/cofy-x/elyro/internal/version"
)

const DefaultPrefix = "ghcr.io/cofy-x/elyro"

func Prefix() string {
	prefix := strings.TrimSpace(os.Getenv("ELYRO_IMAGE_PREFIX"))
	if prefix == "" {
		prefix = DefaultPrefix
	}
	return strings.TrimRight(prefix, "/")
}

func DefaultPlatform() string {
	switch runtime.GOARCH {
	case "amd64":
		return "linux/amd64"
	case "arm64":
		return "linux/arm64"
	default:
		return "linux/" + runtime.GOARCH
	}
}

func Reference(repo, platform string) string {
	name := strings.TrimPrefix(strings.TrimSpace(repo), "elyro/")
	if name == "" {
		return ""
	}
	tag := elyroversion.ReleaseVersion()
	if !elyroversion.IsRelease() {
		arch := PlatformArch(platform)
		if arch == "" {
			return ""
		}
		tag = "dev-" + arch
	}
	return fmt.Sprintf("%s/%s:%s", Prefix(), name, tag)
}

func PlatformArch(platform string) string {
	switch strings.TrimSpace(platform) {
	case "linux/amd64":
		return "amd64"
	case "linux/arm64":
		return "arm64"
	default:
		return ""
	}
}

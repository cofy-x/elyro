package version

import "strings"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func ReleaseVersion() string {
	value := strings.TrimSpace(Version)
	if value == "" {
		return "dev"
	}
	return value
}

func IsRelease() bool {
	value := ReleaseVersion()
	return value != "dev" && value != "unknown"
}

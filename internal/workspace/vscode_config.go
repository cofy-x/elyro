package workspace

import "maps"

type configuredVSCode struct {
	Extensions []string       `yaml:"extensions"`
	Settings   map[string]any `yaml:"settings"`
}

func mergeSettings(base, overlay map[string]any) map[string]any {
	merged := map[string]any{}
	maps.Copy(merged, base)
	maps.Copy(merged, overlay)
	return merged
}

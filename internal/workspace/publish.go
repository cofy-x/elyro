package workspace

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type PortPublish struct {
	HostPort      int
	ContainerPort int
}

func ParsePublishSpecs(rawSpecs []string) ([]PortPublish, error) {
	publishes := make([]PortPublish, 0, len(rawSpecs))
	for _, raw := range rawSpecs {
		spec := strings.TrimSpace(raw)
		if spec == "" {
			continue
		}

		parts := strings.Split(spec, ":")
		switch len(parts) {
		case 1:
			port, err := parsePort(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid publish %q: %w", raw, err)
			}
			publishes = append(publishes, PortPublish{
				HostPort:      port,
				ContainerPort: port,
			})
		case 2:
			hostPort, err := parsePort(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid publish %q: %w", raw, err)
			}
			containerPort, err := parsePort(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid publish %q: %w", raw, err)
			}
			publishes = append(publishes, PortPublish{
				HostPort:      hostPort,
				ContainerPort: containerPort,
			})
		default:
			return nil, fmt.Errorf("invalid publish %q: use <port> or <host-port>:<container-port>", raw)
		}
	}

	sort.Slice(publishes, func(i, j int) bool {
		if publishes[i].HostPort != publishes[j].HostPort {
			return publishes[i].HostPort < publishes[j].HostPort
		}
		return publishes[i].ContainerPort < publishes[j].ContainerPort
	})
	return publishes, nil
}

func NormalizePublishSpecs(publishes []PortPublish) string {
	parts := make([]string, 0, len(publishes))
	for _, publish := range publishes {
		if publish.HostPort == publish.ContainerPort {
			parts = append(parts, strconv.Itoa(publish.HostPort))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%d", publish.HostPort, publish.ContainerPort))
	}
	return strings.Join(parts, ",")
}

func DockerPublishArgs(publishes []PortPublish) []string {
	args := make([]string, 0, len(publishes)*2)
	for _, publish := range publishes {
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", publish.HostPort, publish.ContainerPort))
	}
	return args
}

func MergePortPublishes(groups ...[]PortPublish) ([]PortPublish, error) {
	merged := make([]PortPublish, 0)
	byHostPort := make(map[int]PortPublish)
	for _, group := range groups {
		for _, publish := range group {
			if existing, ok := byHostPort[publish.HostPort]; ok {
				if existing.ContainerPort == publish.ContainerPort {
					continue
				}
				return nil, fmt.Errorf("host port %d is mapped to both container ports %d and %d", publish.HostPort, existing.ContainerPort, publish.ContainerPort)
			}
			byHostPort[publish.HostPort] = publish
			merged = append(merged, publish)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].HostPort != merged[j].HostPort {
			return merged[i].HostPort < merged[j].HostPort
		}
		return merged[i].ContainerPort < merged[j].ContainerPort
	})
	return merged, nil
}

func parsePort(raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("port must be numeric")
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return port, nil
}

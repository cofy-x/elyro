package cli

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/cofy-x/elyro/internal/workspace"
	dockerruntime "github.com/cofy-x/elyro/internal/workspace/runtime/docker"
)

type workspaceJSONView struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ProjectDir     string   `json:"project_dir"`
	MountDir       string   `json:"mount_dir"`
	Status         string   `json:"status"`
	Environment    string   `json:"environment,omitempty"`
	Toolchain      string   `json:"toolchain,omitempty"`
	Image          string   `json:"image,omitempty"`
	Platform       string   `json:"platform,omitempty"`
	Hostname       string   `json:"hostname,omitempty"`
	PublishedPorts []string `json:"published_ports"`
}

func workspacePayload(project workspace.ProjectContext, info *dockerruntime.Container) workspaceJSONView {
	view := workspaceJSONView{
		ID:             project.ProjectHash,
		Name:           project.Slug,
		ProjectDir:     project.ProjectDir,
		MountDir:       project.MountDir,
		Status:         "absent",
		PublishedPorts: []string{},
	}
	if info == nil {
		return view
	}
	view.Status = info.Status
	view.Environment = strings.TrimSpace(info.Environment)
	view.Toolchain = strings.TrimSpace(info.Toolchain)
	view.Image = strings.TrimSpace(info.Image)
	view.Platform = displayPlatform(info.Platform)
	view.Hostname = strings.TrimSpace(info.Hostname)
	if published := strings.TrimSpace(info.Published); published != "" {
		view.PublishedPorts = strings.Split(published, ",")
	}
	return view
}

func displayOptional(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

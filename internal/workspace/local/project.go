package local

import "github.com/cofy-x/elyro/internal/workspace"

func resolveProject(projectDir, containerName, hostAlias string) (workspace.ProjectContext, error) {
	expanded, err := workspace.ExpandPath(projectDir)
	if err != nil {
		return workspace.ProjectContext{}, err
	}
	return workspace.ResolveProjectContext(expanded, containerName, hostAlias)
}

package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/cofy-x/elyro/internal/workspace/local"
	"github.com/spf13/cobra"
)

type ImageAction string

const ImageActionBuilt ImageAction = "built"

type imageBuildJSONView struct {
	SchemaVersion int                `json:"schema_version"`
	Kind          string             `json:"kind"`
	Action        ImageAction        `json:"action"`
	DurationMS    int64              `json:"duration_ms"`
	Image         imageBuildJSONItem `json:"image"`
}

type imageBuildJSONItem struct {
	Reference   string `json:"reference"`
	ProjectDir  string `json:"project_dir"`
	Environment string `json:"environment"`
	Toolchain   string `json:"toolchain"`
	Platform    string `json:"platform"`
	Context     string `json:"context"`
	Dockerfile  string `json:"dockerfile"`
}

var runImageBuild = func(ctx context.Context, dir string, in io.Reader, logs io.Writer, args ...string) error {
	return runStreamingIO(ctx, dir, in, logs, logs, "docker", args...)
}

var imageBuildWorkspaceStatus = func(ctx context.Context, projectDir string) (bool, error) {
	status, err := local.Status(ctx, local.StatusRequest{ProjectDir: projectDir})
	return err == nil && status.Container != nil, err
}

func newImageBuildCmd() *cobra.Command {
	var projectDir, environmentName, platform string
	var pull, noCache, outputJSON bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the project Workspace image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := workspace.ResolveProjectRoot(projectDir, cmd.Flags().Changed("project-dir"))
			if err != nil {
				return err
			}
			project, err := workspace.ResolveProjectContext(root.Dir, "", "")
			if err != nil {
				return err
			}
			config, err := workspace.LoadProjectImageConfig(root.Dir)
			if err != nil {
				return err
			}
			if config == nil {
				configPath, pathErr := workspace.ProjectConfigPath(root.Dir)
				if pathErr != nil {
					return pathErr
				}
				return fmt.Errorf("workspace config not found: %s; run `elyro image init`", configPath)
			}
			environmentExplicit := cmd.Flags().Changed("environment")
			if !environmentExplicit {
				switch {
				case config.DefaultEnvironment != "":
					environmentName = config.DefaultEnvironment
				case len(config.Environments) == 1:
					for name := range config.Environments {
						environmentName = name
					}
				default:
					return fmt.Errorf("--environment is required when elyro.yaml has multiple Environments and no default")
				}
				environmentExplicit = true
			}
			environment, err := workspace.ResolveEnvironment(root.Dir, project.MountDir, workspace.EnvironmentSelection{
				Environment: environmentName, Platform: platform,
				EnvironmentExplicit: environmentExplicit,
				PlatformExplicit:    cmd.Flags().Changed("platform"),
			})
			if err != nil {
				return err
			}
			if environment.ImageBuild == nil {
				return fmt.Errorf("environment %q has no image build configuration; run `elyro image init`", environment.Name)
			}
			ctx, cancel := signalContext()
			defer cancel()
			workspaceExists, err := imageBuildWorkspaceStatus(ctx, root.Dir)
			if err != nil {
				return fmt.Errorf("inspect Workspace before image build: %w", err)
			}

			args := []string{"build", "--platform", environment.Platform}
			if pull {
				args = append(args, "--pull")
			}
			if noCache {
				args = append(args, "--no-cache")
			}
			if outputJSON || !cliui.New(cmd.ErrOrStderr()).ColorEnabled() {
				args = append(args, "--progress", "plain")
			}
			args = append(args,
				"--file", environment.ImageBuild.Dockerfile,
				"--tag", environment.Image,
				environment.ImageBuild.Context,
			)
			started := time.Now()
			if err := runImageBuild(ctx, root.Dir, cmd.InOrStdin(), cmd.ErrOrStderr(), args...); err != nil {
				return fmt.Errorf("build Workspace image %s: %w", environment.Image, err)
			}
			duration := time.Since(started)
			payload := imageBuildPayload(root.Dir, environment, duration)
			if outputJSON {
				return writeJSON(cmd.OutOrStdout(), payload)
			}
			ui := cliui.New(cmd.OutOrStdout())
			if err := ui.Success(fmt.Sprintf("Workspace image built in %s", formatDuration(duration))); err != nil {
				return err
			}
			if err := ui.Fields(
				cliui.Field{Label: "image", Value: environment.Image},
				cliui.Field{Label: "environment", Value: environment.Name},
				cliui.Field{Label: "toolchain", Value: displayToolchain(string(environment.Toolchain))},
				cliui.Field{Label: "platform", Value: environment.Platform},
				cliui.Field{Label: "dockerfile", Value: environment.ImageBuild.Dockerfile},
			); err != nil {
				return err
			}
			if workspaceExists {
				return ui.Next("elyro up --recreate")
			}
			return ui.Next("elyro up")
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory containing elyro.yaml")
	cmd.Flags().StringVar(&environmentName, "environment", "", "Project Environment to build")
	cmd.Flags().StringVar(&platform, "platform", workspace.DefaultPlatform, "Target image platform (linux/amd64 or linux/arm64)")
	cmd.Flags().BoolVar(&pull, "pull", false, "Always attempt to pull a newer base image")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Build without Docker's layer cache")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print the image result as JSON")
	return cmd
}

func imageBuildPayload(projectDir string, environment workspace.ResolvedEnvironment, duration time.Duration) imageBuildJSONView {
	return imageBuildJSONView{
		SchemaVersion: 1, Kind: "image", Action: ImageActionBuilt, DurationMS: duration.Milliseconds(),
		Image: imageBuildJSONItem{
			Reference: environment.Image, ProjectDir: projectDir, Environment: environment.Name,
			Toolchain: string(environment.Toolchain), Platform: environment.Platform,
			Context: environment.ImageBuild.Context, Dockerfile: environment.ImageBuild.Dockerfile,
		},
	}
}

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cofy-x/elyro/internal/cliui"
	"github.com/cofy-x/elyro/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const projectImageDockerfile = ".elyro/Dockerfile"

func newImageInitCmd() *cobra.Command {
	var projectDir, environmentName, toolchainName, imageName string
	var yes bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a project Workspace image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := workspace.ResolveProjectRoot(projectDir, cmd.Flags().Changed("project-dir"))
			if err != nil {
				return err
			}
			return initProjectImage(imageInitOptions{
				ProjectDir: root.Dir, Environment: strings.TrimSpace(environmentName),
				Toolchain: strings.TrimSpace(toolchainName), Image: strings.TrimSpace(imageName),
				Yes: yes, Interactive: isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()),
				In: cmd.InOrStdin(), Out: cmd.OutOrStdout(),
			})
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory to configure")
	cmd.Flags().StringVar(&environmentName, "environment", "", "Project Environment to configure")
	cmd.Flags().StringVar(&toolchainName, "toolchain", "", "Official Toolchain base image (python, go, java, or node)")
	cmd.Flags().StringVar(&imageName, "image", "", "Tagged project image reference")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Create the image configuration without confirmation")
	return cmd
}

type imageInitOptions struct {
	ProjectDir, Environment, Toolchain, Image string
	Yes, Interactive                          bool
	In                                        io.Reader
	Out                                       io.Writer
}

func initProjectImage(options imageInitOptions) error {
	if options.In == nil {
		options.In = strings.NewReader("")
	}
	if options.Out == nil {
		options.Out = io.Discard
	}
	reader := bufferedReader(options.In)
	configPath, err := workspace.ProjectConfigPath(options.ProjectDir)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(configPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("workspace config is not a regular file: %s", configPath)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", configPath, err)
	}
	config, err := workspace.LoadProjectImageConfig(options.ProjectDir)
	if err != nil {
		return err
	}
	if config != nil {
		if err := workspace.ValidateProjectConfiguration(options.ProjectDir); err != nil {
			return err
		}
	}
	environmentName, existing, err := selectImageEnvironment(config, options, reader, options.Out)
	if err != nil {
		return err
	}
	if existing.Platform != "" {
		if err := workspace.ValidatePlatform(existing.Platform); err != nil {
			return fmt.Errorf("environment %q: %w", environmentName, err)
		}
	}
	if existing.HasBuild {
		return fmt.Errorf("environment %q already has image build configuration", environmentName)
	}

	toolchainName := options.Toolchain
	if existing.Toolchain != "" {
		if toolchainName != "" && toolchainName != existing.Toolchain {
			return fmt.Errorf("environment %q already uses Toolchain %s", environmentName, existing.Toolchain)
		}
		toolchainName = existing.Toolchain
	}
	var toolchain workspace.Toolchain
	if toolchainName == "" {
		toolchain, err = initToolchain(options.ProjectDir, "", options.Interactive, reader, options.Out)
	} else {
		toolchain, err = workspace.ParseToolchain(toolchainName)
	}
	if err != nil {
		return err
	}

	imageName := options.Image
	if existing.Image != "" {
		if imageName != "" && imageName != existing.Image {
			return fmt.Errorf("environment %q already uses image %s", environmentName, existing.Image)
		}
		imageName = existing.Image
	}
	if imageName == "" {
		if !options.Interactive {
			return errors.New("--image is required in a non-interactive session")
		}
		imageName = defaultProjectImage(options.ProjectDir)
		if !options.Yes {
			imageName, err = promptImageName(reader, options.Out, imageName)
			if err != nil {
				return err
			}
		}
	}
	if err := workspace.ValidateBuildImageReference(imageName); err != nil {
		return err
	}

	dockerfilePath := filepath.Join(options.ProjectDir, filepath.FromSlash(projectImageDockerfile))
	if _, err := os.Lstat(dockerfilePath); err == nil {
		return fmt.Errorf("Dockerfile already exists: %s", dockerfilePath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", dockerfilePath, err)
	}
	configData, mode, err := imageConfigData(configPath, config, environmentName, toolchain, imageName)
	if err != nil {
		return err
	}
	dockerfileData := []byte(projectDockerfile(toolchain.Image(imageInitPlatform(existing.Platform))))
	if !options.Yes {
		if !options.Interactive {
			return errors.New("refusing to write project image files in a non-interactive session; pass --yes")
		}
		confirmed, err := promptConfirmation(reader, options.Out, fmt.Sprintf("Create %s and update elyro.yaml?", projectImageDockerfile))
		if err != nil {
			return err
		}
		if !confirmed {
			return cliui.New(options.Out).Warning("Project image initialization cancelled")
		}
	}
	if err := installProjectImageFiles(options.ProjectDir, configPath, config == nil, mode, configData, dockerfilePath, dockerfileData); err != nil {
		return err
	}
	ui := cliui.New(options.Out)
	if err := ui.Success("Project Workspace image configured"); err != nil {
		return err
	}
	if err := ui.Fields(
		cliui.Field{Label: "config", Value: configPath},
		cliui.Field{Label: "environment", Value: environmentName},
		cliui.Field{Label: "toolchain", Value: string(toolchain)},
		cliui.Field{Label: "base image", Value: toolchain.Image(imageInitPlatform(existing.Platform))},
		cliui.Field{Label: "target image", Value: imageName},
		cliui.Field{Label: "dockerfile", Value: projectImageDockerfile},
	); err != nil {
		return err
	}
	buildCommand := "elyro image build"
	if config != nil && environmentName != config.DefaultEnvironment {
		buildCommand += " --environment " + strconv.Quote(environmentName)
	}
	return ui.Next("Edit "+projectImageDockerfile, buildCommand)
}

func selectImageEnvironment(config *workspace.ProjectImageConfig, options imageInitOptions, in io.Reader, out io.Writer) (string, workspace.ProjectImageEnvironment, error) {
	if config == nil {
		name := options.Environment
		if name == "" {
			name = "dev"
		}
		return name, workspace.ProjectImageEnvironment{}, nil
	}
	if options.Environment != "" {
		environment, ok := config.Environments[options.Environment]
		if !ok {
			return "", workspace.ProjectImageEnvironment{}, fmt.Errorf("environment %q not found in elyro.yaml", options.Environment)
		}
		return options.Environment, environment, nil
	}
	if config.DefaultEnvironment != "" {
		environment, ok := config.Environments[config.DefaultEnvironment]
		if !ok {
			return "", workspace.ProjectImageEnvironment{}, fmt.Errorf("default environment %q not found in elyro.yaml", config.DefaultEnvironment)
		}
		return config.DefaultEnvironment, environment, nil
	}
	if len(config.Environments) == 1 {
		for name, environment := range config.Environments {
			return name, environment, nil
		}
	}
	if !options.Interactive {
		return "", workspace.ProjectImageEnvironment{}, errors.New("--environment is required when elyro.yaml has multiple Environments and no default")
	}
	names := make([]string, 0, len(config.Environments))
	for name := range config.Environments {
		names = append(names, name)
	}
	sort.Strings(names)
	ui := cliui.New(out)
	if err := ui.Question("Choose an Environment"); err != nil {
		return "", workspace.ProjectImageEnvironment{}, err
	}
	for i, name := range names {
		fmt.Fprintf(out, "  %d  %s\n", i+1, name)
	}
	if err := ui.Prompt("Select: "); err != nil {
		return "", workspace.ProjectImageEnvironment{}, err
	}
	line, err := bufferedReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", workspace.ProjectImageEnvironment{}, err
	}
	choice := strings.TrimSpace(line)
	for i, name := range names {
		if choice == name || choice == fmt.Sprintf("%d", i+1) {
			return name, config.Environments[name], nil
		}
	}
	return "", workspace.ProjectImageEnvironment{}, errors.New("invalid Environment selection; pass --environment <name>")
}

func promptImageName(in io.Reader, out io.Writer, defaultName string) (string, error) {
	ui := cliui.New(out)
	if err := ui.Prompt(fmt.Sprintf("Image [%s]: ", defaultName)); err != nil {
		return "", err
	}
	line, err := bufferedReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if value := strings.TrimSpace(line); value != "" {
		return value, nil
	}
	return defaultName, nil
}

func defaultProjectImage(projectDir string) string {
	name := strings.ToLower(filepath.Base(filepath.Clean(projectDir)))
	var result strings.Builder
	lastDash := false
	for _, r := range name {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-'
		if valid {
			result.WriteRune(r)
			lastDash = r == '-'
		} else if !lastDash && result.Len() > 0 {
			result.WriteByte('-')
			lastDash = true
		}
	}
	sanitized := strings.Trim(result.String(), ".-_")
	if sanitized == "" {
		sanitized = "project"
	}
	return "elyro-local/" + sanitized + ":dev"
}

func imageConfigData(configPath string, config *workspace.ProjectImageConfig, environment string, toolchain workspace.Toolchain, image string) ([]byte, os.FileMode, error) {
	mode := os.FileMode(0o644)
	var document yaml.Node
	if config == nil {
		document = newImageConfigDocument(environment, toolchain, image)
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, 0, fmt.Errorf("read %s: %w", configPath, err)
		}
		info, err := os.Stat(configPath)
		if err != nil {
			return nil, 0, fmt.Errorf("inspect %s: %w", configPath, err)
		}
		mode = info.Mode().Perm()
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, 0, fmt.Errorf("parse %s: %w", configPath, err)
		}
		if err := addImageBuildToDocument(&document, environment, toolchain, image); err != nil {
			return nil, 0, err
		}
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, 0, fmt.Errorf("encode elyro.yaml: %w", err)
	}
	return output.Bytes(), mode, nil
}

func newImageConfigDocument(environment string, toolchain workspace.Toolchain, image string) yaml.Node {
	environmentNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	yamlAppendScalar(environmentNode, "toolchain", string(toolchain))
	yamlAppendScalar(environmentNode, "image", image)
	build := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	yamlAppendScalar(build, "context", ".")
	yamlAppendScalar(build, "dockerfile", projectImageDockerfile)
	environmentNode.Content = append(environmentNode.Content, yamlScalar("build"), build)
	environments := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{yamlScalar(environment), environmentNode}}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	yamlAppendScalar(root, "version", "1")
	root.Content[len(root.Content)-1].Tag = "!!int"
	yamlAppendScalar(root, "default_environment", environment)
	root.Content = append(root.Content, yamlScalar("environments"), environments)
	return yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
}

func addImageBuildToDocument(document *yaml.Node, environment string, toolchain workspace.Toolchain, image string) error {
	root := yamlDocumentMapping(document)
	environments := yamlMappingValue(root, "environments")
	if environments == nil || environments.Kind != yaml.MappingNode {
		return errors.New("elyro.yaml environments must be a mapping")
	}
	target := yamlMappingValue(environments, environment)
	if target == nil || target.Kind != yaml.MappingNode {
		return fmt.Errorf("environment %q is not a mapping", environment)
	}
	toolchainNode := yamlMappingValue(target, "toolchain")
	if toolchainNode == nil {
		yamlAppendScalar(target, "toolchain", string(toolchain))
	} else if strings.TrimSpace(toolchainNode.Value) == "" {
		toolchainNode.Kind, toolchainNode.Tag, toolchainNode.Value = yaml.ScalarNode, "!!str", string(toolchain)
	}
	imageNode := yamlMappingValue(target, "image")
	if imageNode == nil {
		yamlAppendScalar(target, "image", image)
	} else if strings.TrimSpace(imageNode.Value) == "" {
		imageNode.Kind, imageNode.Tag, imageNode.Value = yaml.ScalarNode, "!!str", image
	}
	build := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	yamlAppendScalar(build, "context", ".")
	yamlAppendScalar(build, "dockerfile", projectImageDockerfile)
	if yamlMappingValue(target, "build") != nil {
		return fmt.Errorf("environment %q already has image build configuration", environment)
	}
	target.Content = append(target.Content, yamlScalar("build"), build)
	return nil
}

func yamlDocumentMapping(document *yaml.Node) *yaml.Node {
	if document != nil && document.Kind == yaml.DocumentNode && len(document.Content) == 1 {
		return document.Content[0]
	}
	return nil
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func yamlAppendScalar(mapping *yaml.Node, key, value string) {
	mapping.Content = append(mapping.Content,
		yamlScalar(key),
		yamlScalar(value),
	)
}

func yamlScalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func projectDockerfile(baseImage string) string {
	return fmt.Sprintf("FROM %s\n\n# Add project-wide OS tools here. For example:\n# RUN apt-get update \\\n#     && apt-get install -y --no-install-recommends sqlite3 \\\n#     && rm -rf /var/lib/apt/lists/*\n", baseImage)
}

func imageInitPlatform(configured string) string {
	if configured != "" {
		return configured
	}
	return workspace.DefaultPlatform
}

func installProjectImageFiles(projectDir, configPath string, newConfig bool, configMode os.FileMode, configData []byte, dockerfilePath string, dockerfileData []byte) error {
	elyroDir := filepath.Dir(dockerfilePath)
	createdDir := false
	committed := false
	if _, err := os.Stat(elyroDir); os.IsNotExist(err) {
		if err := os.Mkdir(elyroDir, 0o755); err != nil {
			return fmt.Errorf("create .elyro directory: %w", err)
		}
		createdDir = true
	} else if err != nil {
		return fmt.Errorf("inspect .elyro directory: %w", err)
	}
	defer func() {
		if createdDir && !committed {
			_ = os.Remove(elyroDir)
		}
	}()
	if err := validateImageInitDirectory(projectDir, elyroDir); err != nil {
		if createdDir {
			_ = os.Remove(elyroDir)
		}
		return err
	}
	if !newConfig {
		info, err := os.Lstat(configPath)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", configPath, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("workspace config is not a regular file: %s", configPath)
		}
	}
	dockerTemp, err := os.CreateTemp(elyroDir, ".Dockerfile-*")
	if err != nil {
		return fmt.Errorf("create temporary Dockerfile: %w", err)
	}
	dockerTempPath := dockerTemp.Name()
	defer os.Remove(dockerTempPath)
	configTemp, err := os.CreateTemp(projectDir, ".elyro.yaml-*")
	if err != nil {
		dockerTemp.Close()
		return fmt.Errorf("create temporary workspace config: %w", err)
	}
	configTempPath := configTemp.Name()
	defer os.Remove(configTempPath)
	if err := writeStagedFile(dockerTemp, 0o644, dockerfileData); err != nil {
		configTemp.Close()
		return err
	}
	if err := writeStagedFile(configTemp, configMode, configData); err != nil {
		return err
	}
	if err := os.Rename(dockerTempPath, dockerfilePath); err != nil {
		return fmt.Errorf("install %s: %w", dockerfilePath, err)
	}
	installConfig := func() error { return os.Rename(configTempPath, configPath) }
	if newConfig {
		installConfig = func() error { return os.Link(configTempPath, configPath) }
	}
	if err := installConfig(); err != nil {
		_ = os.Remove(dockerfilePath)
		if createdDir {
			_ = os.Remove(elyroDir)
		}
		return fmt.Errorf("install %s: %w", configPath, err)
	}
	committed = true
	return nil
}

func validateImageInitDirectory(projectDir, targetDir string) error {
	root, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}
	target, err := filepath.EvalSymlinks(targetDir)
	if err != nil {
		return fmt.Errorf("resolve .elyro directory: %w", err)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New(".elyro directory resolves outside the project")
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("inspect .elyro directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New(".elyro path is not a directory")
	}
	return nil
}

func writeStagedFile(file *os.File, mode os.FileMode, data []byte) error {
	name := file.Name()
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return fmt.Errorf("set temporary file permissions: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary file %s: %w", name, err)
	}
	return nil
}

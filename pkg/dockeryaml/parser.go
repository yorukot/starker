package dockeryaml

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// ParseComposeContent parses Docker Compose YAML content and returns a ComposeFile
func ParseComposeContent(yamlContent string, projectName string) (*types.Project, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("compose content cannot be empty")
	}

	configFiles := []types.ConfigFile{
		{
			Filename: "docker-compose.yml",
			Content:  []byte(yamlContent),
		},
	}

	project, err := loader.LoadWithContext(context.Background(), types.ConfigDetails{
		ConfigFiles: configFiles,
		WorkingDir:  ".",
	}, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return project, nil
}

func Validate(project *types.Project) error {
	if project == nil {
		return fmt.Errorf("project is nil")
	}

	if len(project.Services) == 0 {
		return fmt.Errorf("compose file must contain at least one service")
	}

	for _, service := range project.Services {
		if service.Name == "" {
			return fmt.Errorf("service name cannot be empty")
		}
		if service.Image == "" && service.Build == nil {
			return fmt.Errorf("service '%s' must specify either image or build", service.Name)
		}
	}

	return nil
}

package utils

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"

	"github.com/yorukot/starker/pkg/generator"
)

// createProjectVolumes creates volumes defined in the compose project
func createProjectVolumes(ctx context.Context, dockerClient *client.Client, project *types.Project, streamResult *StreamingResult, namingGen *generator.NamingGenerator) error {
	for volumeName, volumeConfig := range project.Volumes {
		fullVolumeName := volumeConfig.Name
		if fullVolumeName == "" {
			fullVolumeName = namingGen.VolumeName(volumeName)
		}

		// Check if volume already exists
		existingVolumes, err := dockerClient.VolumeList(ctx, volume.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", fullVolumeName)),
		})
		if err != nil {
			return fmt.Errorf("failed to list volumes: %w", err)
		}

		if len(existingVolumes.Volumes) == 0 {
			streamResult.LogChan <- fmt.Sprintf("Creating volume: %s", fullVolumeName)
			_, err = dockerClient.VolumeCreate(ctx, volume.CreateOptions{
				Name:   fullVolumeName,
				Labels: namingGen.GetVolumeLabels(project.Name, volumeName),
			})
			if err != nil {
				return fmt.Errorf("failed to create volume %s: %w", fullVolumeName, err)
			}
			streamResult.LogChan <- fmt.Sprintf("Created volume: %s", fullVolumeName)
		} else {
			streamResult.LogChan <- fmt.Sprintf("Volume %s already exists", fullVolumeName)
		}
	}

	return nil
}

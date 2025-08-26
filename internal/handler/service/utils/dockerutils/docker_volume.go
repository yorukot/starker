package dockerutils

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
func createProjectVolume(ctx context.Context, dockerClient *client.Client, volumeConfig types.VolumeConfig, streamResult *StreamingResult, namingGen *generator.NamingGenerator) error {
	fullVolumeName := namingGen.VolumeName(volumeConfig.Name)

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
			Labels: namingGen.GetVolumeLabels(namingGen.ProjectName(), volumeConfig.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to create volume %s: %w", fullVolumeName, err)
		}
		streamResult.LogChan <- fmt.Sprintf("Created volume: %s", fullVolumeName)
	} else {
		streamResult.LogChan <- fmt.Sprintf("Volume %s already exists", fullVolumeName)
	}

	return nil
}

package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	dockervolume "github.com/docker/docker/api/types/volume"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
)

func (dh *DockerHandler) StartDockerVolumes(ctx context.Context, tx pgx.Tx) error {

	for _, volume := range dh.Project.Volumes {
		// Generate the docker volume name and create the Docker volume
		volumeID, err := dh.StartDockerVolume(ctx, volume)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to start docker volume %s: %v", volume.Name, err),
			}
			return err
		}

		// Create the volume record in database
		serviceVolume := models.ServiceVolume{
			ID:         ksuid.New().String(),
			ServiceID:  dh.NamingGenerator.ServiceID(),
			VolumeID:   &volumeID,
			VolumeName: dh.NamingGenerator.VolumeName(volume.Name),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err = repository.CreateServiceVolume(ctx, tx, serviceVolume)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to save volume to database: %v", err),
			}
			return fmt.Errorf("failed to save volume %s to database: %w", volume.Name, err)
		}

		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogTypeInfo,
			Message: fmt.Sprintf("Volume %s created and saved successfully", volume.Name),
		}
	}
	return nil
}

// StartDockerVolume creates a Docker volume and returns the volume ID
func (dh *DockerHandler) StartDockerVolume(ctx context.Context, volumeConfig types.VolumeConfig) (volumeID string, err error) {
	// Generate volume name using naming generator
	volumeName := dh.NamingGenerator.VolumeName(volumeConfig.Name)

	// Generate project name and labels
	projectName := dh.NamingGenerator.ProjectName()
	labels := dh.NamingGenerator.GetVolumeLabels(projectName, volumeConfig.Name)

	// Log volume creation start
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogStep,
		Message: fmt.Sprintf("Creating Docker volume: %s", volumeName),
	}

	// Prepare volume creation options
	createOptions := dockervolume.CreateOptions{
		Name:       volumeName,
		Labels:     labels,
		Driver:     volumeConfig.Driver,
		DriverOpts: volumeConfig.DriverOpts,
	}

	// Create the Docker volume
	dockerVolume, err := dh.Client.VolumeCreate(ctx, createOptions)
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to create Docker volume %s: %v", volumeName, err),
		}
		return "", fmt.Errorf("failed to create Docker volume %s: %w", volumeName, err)
	}

	// Log successful creation
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogTypeInfo,
		Message: fmt.Sprintf("Successfully created Docker volume: %s", dockerVolume.Name),
	}

	return dockerVolume.Name, nil
}

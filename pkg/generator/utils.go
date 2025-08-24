package generator

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockerpool"
)

type DockerConfig struct {
	Generator   *NamingGenerator
	Client      any
	ProjectName string
}

func PrepareDockerConfigForService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dockerPool *dockerpool.DockerConnectionPool) (*DockerConfig, error) {
	service, err := repository.GetServiceByID(ctx, db, serviceID, teamID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	server, err := repository.GetServerByID(ctx, db, service.ServerID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	privateKey, err := repository.GetPrivateKeyByID(ctx, db, server.PrivateKeyID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	generator := NewNamingGenerator(serviceID, teamID, server.ID)

	host := fmt.Sprintf("ssh://%s@%s:%s", server.User, server.IP, server.Port)

	connectionID := generator.ConnectionID()
	dockerClient, err := dockerPool.GetConnection(connectionID, host, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker connection: %w", err)
	}

	return &DockerConfig{
		Generator:   generator,
		Client:      dockerClient,
		ProjectName: generator.ProjectName(),
	}, nil
}

func (dc *DockerConfig) GetServiceFilters(serviceName string) filters.Args {
	fb := NewFilterBuilder(dc.Generator)
	return fb.ServiceFilters(dc.ProjectName, serviceName)
}

func (dc *DockerConfig) GetProjectFilters() filters.Args {
	fb := NewFilterBuilder(dc.Generator)
	return fb.ProjectFilters(dc.ProjectName)
}

func (dc *DockerConfig) GetNetworkFilters() filters.Args {
	fb := NewFilterBuilder(dc.Generator)
	return fb.NetworkFilters(dc.ProjectName)
}

func (dc *DockerConfig) GetVolumeFilters() filters.Args {
	fb := NewFilterBuilder(dc.Generator)
	return fb.VolumeFilters(dc.ProjectName)
}

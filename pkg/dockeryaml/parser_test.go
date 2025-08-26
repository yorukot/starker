package dockeryaml

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposeContent(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		projectName string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, project *types.Project)
	}{
		{
			name:        "empty content",
			yamlContent: "",
			projectName: "test-project",
			wantErr:     true,
			errContains: "compose content cannot be empty",
		},
		{
			name:        "empty project name",
			yamlContent: "services:\n  web:\n    image: nginx",
			projectName: "",
			wantErr:     true,
			errContains: "project name must not be empty",
		},
		{
			name: "valid simple service",
			yamlContent: `services:
  web:
    image: nginx:latest
    ports:
      - "80:80"`,
			projectName: "test-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				assert.NotNil(t, project)
				assert.Equal(t, "test-project", project.Name)
				assert.Len(t, project.Services, 1)

				service := project.Services["web"]
				assert.Equal(t, "web", service.Name)
				assert.Equal(t, "nginx:latest", service.Image)
				assert.Len(t, service.Ports, 1)
				assert.Equal(t, uint32(80), service.Ports[0].Target)
			},
		},
		{
			name: "multiple services",
			yamlContent: `services:
  web:
    image: nginx:latest
    depends_on:
      - db
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb`,
			projectName: "multi-service",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				assert.Equal(t, "multi-service", project.Name)
				assert.Len(t, project.Services, 2)

				web := project.Services["web"]
				assert.Equal(t, "nginx:latest", web.Image)
				assert.Contains(t, web.DependsOn, "db")

				db := project.Services["db"]
				assert.Equal(t, "postgres:13", db.Image)
				assert.Equal(t, "testdb", *db.Environment["POSTGRES_DB"])
			},
		},
		{
			name: "service with build context",
			yamlContent: `services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3000:3000"`,
			projectName: "build-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				service := project.Services["app"]
				assert.NotNil(t, service.Build)
				assert.Equal(t, ".", service.Build.Context)
				assert.Equal(t, "Dockerfile", service.Build.Dockerfile)
			},
		},
		{
			name: "service with volumes",
			yamlContent: `services:
  app:
    image: node:16
    volumes:
      - ./src:/app/src
      - node_modules:/app/node_modules
volumes:
  node_modules:`,
			projectName: "volume-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				service := project.Services["app"]
				assert.Len(t, service.Volumes, 2)
				assert.Len(t, project.Volumes, 1)
			},
		},
		{
			name: "service with networks",
			yamlContent: `services:
  web:
    image: nginx
    networks:
      - frontend
      - backend
networks:
  frontend:
  backend:`,
			projectName: "network-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				service := project.Services["web"]
				assert.Len(t, service.Networks, 2)
				assert.Len(t, project.Networks, 2)
			},
		},
		{
			name: "invalid yaml syntax",
			yamlContent: `services:
  web:
    image: nginx
    ports
      - "80:80"`, // Missing colon after ports
			projectName: "invalid-project",
			wantErr:     true,
			errContains: "failed to parse compose file",
		},
		{
			name: "compose with version (ignored)",
			yamlContent: `version: "3.8"
services:
  web:
    image: nginx`,
			projectName: "version-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				assert.Equal(t, "version-project", project.Name)
				assert.Len(t, project.Services, 1)
			},
		},
		{
			name: "service with environment variables",
			yamlContent: `services:
  app:
    image: node:16
    environment:
      - NODE_ENV=production
      - PORT=3000
      - DEBUG=true`,
			projectName: "env-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				service := project.Services["app"]
				assert.Equal(t, "production", *service.Environment["NODE_ENV"])
				assert.Equal(t, "3000", *service.Environment["PORT"])
				assert.Equal(t, "true", *service.Environment["DEBUG"])
			},
		},
		{
			name: "service with restart policy",
			yamlContent: `services:
  app:
    image: redis:alpine
    restart: unless-stopped`,
			projectName: "restart-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				service := project.Services["app"]
				assert.Equal(t, "unless-stopped", service.Restart)
			},
		},
		{
			name: "complex compose file",
			yamlContent: `services:
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev
    ports:
      - "3000:3000"
    volumes:
      - ./frontend:/app
      - /app/node_modules
    environment:
      - REACT_APP_API_URL=http://backend:5000
    depends_on:
      - backend
    networks:
      - app-network

  backend:
    build:
      context: ./backend
    ports:
      - "5000:5000"
    environment:
      DATABASE_URL: postgresql://user:pass@db:5432/myapp
    depends_on:
      - db
    networks:
      - app-network

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - app-network

volumes:
  postgres_data:

networks:
  app-network:
    driver: bridge`,
			projectName: "complex-project",
			wantErr:     false,
			validate: func(t *testing.T, project *types.Project) {
				assert.Equal(t, "complex-project", project.Name)
				assert.Len(t, project.Services, 3)
				assert.Len(t, project.Volumes, 1)
				assert.Len(t, project.Networks, 1)

				// Validate frontend service
				frontend := project.Services["frontend"]
				assert.NotNil(t, frontend.Build)
				assert.Equal(t, "frontend", frontend.Build.Context)
				assert.Contains(t, frontend.DependsOn, "backend")

				// Validate backend service
				backend := project.Services["backend"]
				assert.Contains(t, backend.DependsOn, "db")
				assert.Equal(t, "postgresql://user:pass@db:5432/myapp", *backend.Environment["DATABASE_URL"])

				// Validate database service
				db := project.Services["db"]
				assert.Equal(t, "postgres:13", db.Image)
				assert.Equal(t, "myapp", *db.Environment["POSTGRES_DB"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := ParseComposeContent(tt.yamlContent, tt.projectName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, project)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				if tt.validate != nil {
					tt.validate(t, project)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		project     *types.Project
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil project",
			project:     nil,
			wantErr:     true,
			errContains: "project is nil",
		},
		{
			name: "empty services",
			project: &types.Project{
				Name:     "empty-project",
				Services: types.Services{},
			},
			wantErr:     true,
			errContains: "compose file must contain at least one service",
		},
		{
			name: "service with empty name",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"": {
						Name:  "",
						Image: "nginx",
					},
				},
			},
			wantErr:     true,
			errContains: "service name cannot be empty",
		},
		{
			name: "service without image or build",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name: "web",
						// No Image or Build specified
					},
				},
			},
			wantErr:     true,
			errContains: "service 'web' must specify either image or build",
		},
		{
			name: "valid service with image",
			project: &types.Project{
				Name: "valid-project",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid service with build",
			project: &types.Project{
				Name: "valid-build-project",
				Services: types.Services{
					"app": {
						Name: "app",
						Build: &types.BuildConfig{
							Context: ".",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid services",
			project: &types.Project{
				Name: "multi-service-project",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
					"api": {
						Name: "api",
						Build: &types.BuildConfig{
							Context: "./api",
						},
					},
					"db": {
						Name:  "db",
						Image: "postgres:13",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid services",
			project: &types.Project{
				Name: "mixed-project",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
					"invalid": {
						Name: "invalid",
						// No image or build
					},
				},
			},
			wantErr:     true,
			errContains: "service 'invalid' must specify either image or build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.project)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAndValidate(t *testing.T) {
	t.Run("parse valid content and validate", func(t *testing.T) {
		yamlContent := `services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb`

		project, err := ParseComposeContent(yamlContent, "integration-test")
		require.NoError(t, err)
		require.NotNil(t, project)

		err = Validate(project)
		assert.NoError(t, err)
	})

	t.Run("parse invalid content that fails at parse time", func(t *testing.T) {
		// This creates a service without image or build, which should fail parsing
		yamlContent := `services:
  invalid-service:
    ports:
      - "80:80"`

		project, err := ParseComposeContent(yamlContent, "validation-fail-test")
		assert.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "failed to parse compose file")
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("whitespace only content", func(t *testing.T) {
		project, err := ParseComposeContent("   \n  \t  \n  ", "whitespace-test")
		assert.Error(t, err)
		assert.Nil(t, project)
	})

	t.Run("only version specified", func(t *testing.T) {
		yamlContent := `version: "3.8"`
		project, err := ParseComposeContent(yamlContent, "version-only")
		assert.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "empty compose file")
	})

	t.Run("service with both image and build", func(t *testing.T) {
		yamlContent := `services:
  app:
    image: nginx:latest
    build:
      context: .`

		project, err := ParseComposeContent(yamlContent, "both-image-build")
		require.NoError(t, err)

		err = Validate(project)
		assert.NoError(t, err) // Both image and build is valid, build takes precedence
	})

	t.Run("very long project name", func(t *testing.T) {
		longName := "very-long-project-name-that-might-cause-issues-in-some-systems-or-docker-implementations"
		yamlContent := `services:
  web:
    image: nginx`

		project, err := ParseComposeContent(yamlContent, longName)
		require.NoError(t, err)
		assert.Equal(t, longName, project.Name)
	})
}

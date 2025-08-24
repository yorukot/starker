# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build & Run
- `make build` - Build the binary to `tmp/starker`
- `make run` - Build and run the application
- `make dev` - Run with hot reload using Air
- `make test` - Run all tests with `go test ./...`
- `make clean` - Remove the `tmp/` directory

### Code Quality
- `make lint` - Run Go formatting, vet, and golint
  - `go fmt ./...` - Format code
  - `go vet ./...` - Vet code for issues
  - `golint ./...` - Lint code style

### Documentation
- `make generate-docs` - Generate Swagger documentation using swag

### Database
- Start PostgreSQL: `docker compose up postgres -d`
- Database migrations are in `migrations/` directory

## Project Architecture

### Core Structure
This is a Go web API using Chi router with a PostgreSQL database, following a clean architecture pattern:

**Main Components:**
- `cmd/main.go` - Entry point with server setup and routing
- `internal/config/` - Environment configuration using caarlos0/env
- `internal/database/` - PostgreSQL connection using pgx/v5
- `internal/handler/` - HTTP handlers organized by domain (auth, team, privatekey, server, service, project)
- `internal/middleware/` - Custom middleware (auth, logging)
- `internal/models/` - Data models (User, Account, OAuthToken, Team, Server, PrivateKey, Service, Project)
- `internal/repository/` - Database access layer
- `internal/service/` - Business logic layer (authsvc, teamsvc, privatekeysvc, servicesvc)
- `internal/router/` - Route definitions by domain
- `pkg/` - Reusable packages (encrypt, logger, response, dockerpool, dockeryaml, generator)

### Authentication System
The application implements OAuth2 (Google) and JWT-based authentication:
- OAuth state management with configurable expiration
- JWT access tokens (15 min default) and refresh tokens (365 days default)
- Password hashing using Argon2
- Account-based authentication supporting multiple providers

### Database Schema
- `users` - User profiles with display name, avatar
- `accounts` - Authentication accounts linked to users (email, OAuth)
- `oauth_tokens` - OAuth provider tokens
- `refresh_tokens` - JWT refresh tokens with metadata
- `teams` / `team_users` / `team_invites` - Team management system
- `servers` - Server configurations with SSH connection details
- `private_keys` - SSH private keys for server authentication
- `projects` - Project containers for organizing services
- `services` - Docker Compose services with state management
- `service_compose_configs` - Docker Compose file configurations for services

### Key Dependencies
- **Router:** `go-chi/chi/v5` for HTTP routing
- **Database:** `jackc/pgx/v5` for PostgreSQL
- **Auth:** `golang-jwt/jwt/v5`, `coreos/go-oidc/v3`
- **Config:** `caarlos0/env/v10` for environment variables
- **Logging:** `go.uber.org/zap`
- **Documentation:** `swaggo/swag` for Swagger
- **Migrations:** `golang-migrate/migrate/v4`
- **Docker:** `github.com/docker/docker/client` for Docker API integration
- **Docker Compose:** `github.com/compose-spec/compose-go/v2` and `github.com/docker/compose/v2` for Compose parsing and orchestration

### Docker Service Management Architecture

The platform implements a complete Docker service orchestration system with real-time streaming:

**Core Components:**
- `pkg/dockerpool/` - Connection pooling for Docker API clients over SSH
- `pkg/dockeryaml/` - Docker Compose file parsing and validation
- `pkg/generator/` - Consistent naming and labeling for Docker resources
- `internal/handler/service/utils/` - Docker operation utilities with SSE streaming support

**Service State Management:**
- Services support `start`, `stop`, and `restart` operations via `/teams/{teamID}/projects/{projectID}/services/{serviceID}/state`
- All Docker operations stream real-time progress via Server-Sent Events (SSE)
- Proper dependency orchestration using Docker Compose v2 APIs
- Resource management includes networks, volumes, images, and containers

**Docker Connection Pooling:**
- SSH-based connections to remote Docker daemons with private key authentication
- Connection pooling with idle timeout and connection reuse
- Automatic connection cleanup and resource management
- Generated connection IDs based on service, team, and server identifiers

**Streaming Operations:**
- Real-time progress streaming for image pulls, container starts/stops, and build operations
- `StreamingResult` channels for stdout, stderr, errors, and completion signals
- JSON parsing of Docker API progress events with detailed layer-by-layer updates
- Proper error handling and graceful degradation

### Code Style & Conventions
- **Error Handling:** Use custom error codes and structured error responses via `response.RespondWithError()`
- **Transactions:** Always use database transactions for data modifications with proper rollback handling
- **Logging:** Use structured logging with zap.L() for errors and warnings
- **Documentation:** Include comprehensive Swagger comments for all HTTP endpoints
- **Validation:** Validate request bodies at the handler level using service layer validators
- **Security:** Use secure cookie handling for refresh tokens and proper password hashing
- **Function Organization:** Use visual separators (comment blocks) to organize related functions
- **Repository Function Order:** All repository functions should follow the order: Get → Create → Update → Delete

### Environment Setup
Copy `.env.example` to `.env` and configure:
- JWT secret key
- Google OAuth credentials
- PostgreSQL connection details
- Optional: token expiration settings

### Development Workflow
1. Start PostgreSQL: `docker compose up postgres -d`
2. Run migrations (handled automatically by app)
3. Use `make dev` for hot reload development
4. Access Swagger docs at `/swagger/` in dev mode
5. Health check available at `/health`

### API Structure
- Base path: `/api`
- Authentication endpoints under `/api/auth`
- Team endpoints under `/api/team`
- Private key endpoints under `/api/privatekey`
- Server endpoints under `/api/server`
- Project endpoints under `/api/projects`
- Service endpoints under `/api/teams/{teamID}/projects/{projectID}/services`
- Service state management with SSE streaming at `/teams/{teamID}/projects/{projectID}/services/{serviceID}/state`
- 404/405 handlers with consistent error format

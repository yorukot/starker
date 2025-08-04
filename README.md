# Stargo

A modern Go web API template built with Chi router and PostgreSQL, featuring OAuth 2.0 authentication and clean architecture patterns.

## Architecture

Stargo follows clean architecture principles:

```
internal/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── handler/         # HTTP handlers (controllers)
├── logger/          # Logging configuration
├── middleware/      # HTTP middleware
├── models/          # Data models and types
├── repository/      # Data access layer
├── router/          # Route definitions
├── service/         # Business logic layer
└── utils/           # Utility functions

pkg/
└── encrypt/         # Encryption utilities
```

## Technology Stack

- **Go 1.24.3+** - Programming language
- **Chi v5** - HTTP router
- **PostgreSQL** - Primary database
- **pgx/v5** - PostgreSQL driver
- **golang-migrate** - Database migrations
- **Zap** - Structured logging
- **JWT** - Token-based authentication
- **OAuth2 + OIDC** - Authentication protocols
- **Swagger** - API documentation

## Getting Started

### Prerequisites

- Go 1.24.3 or higher
- PostgreSQL 17+
- Docker and Docker Compose (optional, for local development)

### Environment Variables

Create a `.env` file in the root directory:

```env
# Server Configuration
PORT=8080
APP_ENV=dev

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=stargo
DB_PASSWORD=stargo-this-is-a-really-long-password
DB_NAME=stargo
DB_SSL_MODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-here

# Google OAuth Configuration
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/oauth/google/callback
```

### Installation & Running

```bash
# Clone the repository
git clone https://github.com/yorukot/stargo.git
cd stargo

# Install dependencies
go mod download

# Run the application
go run main.go
```

The server will start on `http://localhost:8080`

### Database Setup

```bash
# Start PostgreSQL container
docker compose up -d postgres
```

### Migrations

Migrations are automatically applied when the application starts. Migration files are located in the `migrations/` directory:

- `1_initialize_schema.up.sql` - Creates all tables and indexes
- `1_initialize_schema.down.sql` - Drops all tables (currently empty)

## API Documentation

The API is documented using Swagger/OpenAPI. After starting the server, you can:

- View generated documentation in `docs/swagger.yaml`
- Access the Swagger UI (if configured)

## Development

### Project Structure

```
internal/
├── config/          # Configuration management
├── database/        # Database connections and migrations
├── handler/         # HTTP handlers (controllers)
├── logger/          # Logging setup
├── middleware/      # HTTP middleware
├── models/          # Data models and type definitions
├── repository/      # Data access layer (DB operations)
├── router/          # Route definitions
├── service/         # Business logic layer
└── utils/           # Utility functions
```

### Adding a New Endpoint

To add a new endpoint, follow these steps:

1. **Define the route**
   Start in `internal/router/router.go` and register your new route.

2. **Create a handler**
   Add your HTTP handler function in `internal/handler/handler.go`. This function should process the incoming request and call the corresponding service function.

3. **Implement the service logic**
   Add your business logic in `internal/service/service.go`. The service is responsible for processing data and coordinating operations.

4. **Use the repository for database operations**
   Inside your service function, use the repository layer to interact with the database. All database access should go through the repository layer.

> Note: The service layer should contain all business logic. It must not directly access the database—always go through the repository.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

**Stargo** - Simple Go API template

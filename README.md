# Stargo

A modern Go web API template built with Chi router and PostgreSQL, featuring OAuth 2.0 authentication and clean architecture patterns.

## Getting Started

### Prerequisites

- Go 1.24.3 or higher
- PostgreSQL 17+
- Docker and Docker Compose (optional, for local development)

### Installation & Running

```bash
# Clone the repository
git clone https://github.com/yorukot/stargo.git
cd stargo

# Install dependencies
go mod download

# Fill the `.env` file with your own values

# Run the application
go run main.go
```

The server will start on `http://localhost:8000` (If you use the default port)

### Database Setup

```bash
# Start PostgreSQL container
docker compose up -d postgres
```

### Migrations

Migrations are automatically applied when the application starts. Migration files are located in the `migrations/` directory:

- `1_initialize_schema.up.sql`
- `1_initialize_schema.down.sql`

> More about migrations: [What are database migrations?](https://www.prisma.io/dataguide/types/relational/what-are-database-migrations)

## API Documentation

The API is documented using Swagger/OpenAPI. After starting the server, you can:

- View generated documentation in `docs/swagger.yaml`
- Access the Swagger UI (if configured)

## Development

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

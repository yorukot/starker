# Stargo

A modern Go web API template built with Chi router and PostgreSQL, featuring OAuth 2.0 authentication and clean architecture patterns.

## 🚀 Features

- **OAuth 2.0 + OpenID Connect** authentication (Google provider included)
- **Clean Architecture** with proper separation of concerns
- **PostgreSQL** database with automated migrations
- **Structured Logging** with Zap
- **API Documentation** with Swagger/OpenAPI
- **Secure Token Management** with JWT and refresh tokens
- **Docker Compose** for local development
- **Middleware System** for request logging and authentication

## 🏗️ Architecture

Stargo follows clean architecture principles:

```
internal/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── handler/         # HTTP handlers (controllers)
├── middleware/      # HTTP middleware
├── models/          # Data models and types
├── repository/      # Data access layer
├── router/          # Route definitions
└── service/         # Business logic layer

pkg/
├── encrypt/         # Encryption utilities
├── logger/          # Logging configuration
└── utils/           # Utility functions
```

## 🛠️ Technology Stack

- **Go 1.24.3+** - Programming language
- **Chi v5** - HTTP router
- **PostgreSQL** - Primary database
- **pgx/v5** - PostgreSQL driver
- **golang-migrate** - Database migrations
- **Zap** - Structured logging
- **JWT** - Token-based authentication
- **OAuth2 + OIDC** - Authentication protocols
- **Swagger** - API documentation

## 🚦 Getting Started

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

### Database Setup

#### Using Docker Compose (Recommended)

```bash
# Start PostgreSQL container
docker-compose up -d postgres

# Verify the database is running
docker-compose ps
```

#### Manual PostgreSQL Setup

1. Install PostgreSQL 17+
2. Create a database named `stargo`
3. Create a user with the credentials from your `.env` file

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

## 📊 Database Schema

The application uses four main tables:

### Users Table
```sql
users (
    id VARCHAR(27) PRIMARY KEY,           -- KSUID identifier
    password TEXT,                        -- Hashed password (nullable)
    avatar TEXT,                          -- Profile picture URL
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
)
```

### Accounts Table
```sql
accounts (
    id VARCHAR(27) PRIMARY KEY,           -- KSUID identifier
    provider VARCHAR(50),                 -- OAuth provider (e.g., 'google')
    provider_user_id VARCHAR(255),        -- Provider's user ID
    user_id VARCHAR(27) REFERENCES users(id),
    email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
)
```

### OAuth Tokens Table
```sql
oauth_tokens (
    account_id VARCHAR(27) PRIMARY KEY REFERENCES accounts(id),
    access_token TEXT,                    -- OAuth access token
    refresh_token TEXT,                   -- OAuth refresh token
    expiry TIMESTAMP WITH TIME ZONE,      -- Token expiration
    token_type VARCHAR(50),               -- Usually 'Bearer'
    provider VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
)
```

### Refresh Tokens Table
```sql
refresh_tokens (
    id VARCHAR(27) PRIMARY KEY,           -- KSUID identifier
    user_id VARCHAR(27) REFERENCES users(id),
    token TEXT UNIQUE,                    -- Secure refresh token
    user_agent TEXT,                      -- Client user agent
    ip INET,                             -- Client IP address
    used_at TIMESTAMP WITH TIME ZONE,     -- When token was used
    created_at TIMESTAMP WITH TIME ZONE
)
```

### Migrations

Migrations are automatically applied when the application starts. Migration files are located in the `migrations/` directory:

- `1_initialize_schema.up.sql` - Creates all tables and indexes
- `1_initialize_schema.down.sql` - Drops all tables (currently empty)

## 🔐 Authentication Flow

### OAuth 2.0 Flow

1. **Initiate OAuth**: `GET /api/auth/oauth/{provider}`
   - Redirects to OAuth provider (e.g., Google)
   - Includes state parameter for CSRF protection

2. **OAuth Callback**: `GET /api/auth/oauth/{provider}/callback`
   - Processes authorization code from provider
   - Verifies ID token using OIDC
   - Creates or updates user account
   - Issues secure refresh token as HTTP-only cookie

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Health check endpoint |
| GET | `/api/auth/oauth/{provider}` | Initiate OAuth flow |
| GET | `/api/auth/oauth/{provider}/callback` | Handle OAuth callback |

### Supported Providers

- ✅ **Google** - Fully implemented
- 🚧 **GitHub** - Coming soon
- 🚧 **Discord** - Coming soon

## 📱 API Documentation

The API is documented using Swagger/OpenAPI. After starting the server, you can:

- View generated documentation in `docs/swagger.yaml`
- Access the Swagger UI (if configured)

### Example OAuth Request

```bash
# Initiate Google OAuth
curl -X GET "http://localhost:8080/api/auth/oauth/google?from=dashboard"

# This will redirect to Google's OAuth consent screen
```

## 🔧 Development

### Project Structure

```
stargo/
├── cmd/                 # Application entry points
│   └── run.go          # Server startup logic
├── docs/               # API documentation
├── internal/           # Private application code
├── migrations/         # Database migration files
├── pkg/               # Public/reusable packages
├── tmp/               # Temporary files (ignored)
├── docker-compose.yml # Local development setup
├── go.mod            # Go module file
└── main.go           # Application entry point
```

### Adding New OAuth Providers

1. **Update the Provider enum** in `internal/models/user.go`:
```go
const (
    ProviderEmail  Provider = "email"
    ProviderGoogle Provider = "google"
    ProviderGitHub Provider = "github"  // Add new provider
)
```

2. **Add provider parsing** in `internal/service/auth.go`:
```go
func ParseProvider(s string) (models.Provider, error) {
    switch s {
    case string(models.ProviderGoogle):
        return models.ProviderGoogle, nil
    case string(models.ProviderGitHub):  // Add new case
        return models.ProviderGitHub, nil
    // ...
    }
}
```

3. **Configure OAuth settings** in `internal/config/oauth.go`:
```go
func OauthConfig() (*OAuthConfig, error) {
    // Add GitHub configuration
    githubOauthConfig := &oauth2.Config{
        RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
        ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
        ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
        Scopes:       []string{"user:email"},
        Endpoint:     github.Endpoint,
    }
    // ...
}
```

### Running in Development

```bash
# Set development environment
export APP_ENV=dev

# Run with hot reload (using air - install with: go install github.com/cosmtrek/air@latest)
air

# Or run normally
go run main.go
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## 🚀 Deployment

### Production Environment Variables

Ensure these are set in production:

```env
APP_ENV=production
PORT=8080
DB_SSL_MODE=require
JWT_SECRET=your-very-secure-jwt-secret-minimum-256-bits
GOOGLE_CLIENT_ID=your-production-google-client-id
GOOGLE_CLIENT_SECRET=your-production-google-client-secret
GOOGLE_REDIRECT_URL=https://yourdomain.com/api/auth/oauth/google/callback
```

### Security Considerations

- ✅ HTTP-only cookies for refresh tokens
- ✅ CSRF protection with state parameters
- ✅ Secure token generation
- ✅ SQL injection protection with pgx
- ✅ Request logging and monitoring
- ✅ Environment-based configuration

### Building for Production

```bash
# Build binary
go build -o stargo main.go

# Run binary
./stargo
```

### Docker Deployment

```dockerfile
FROM golang:1.24.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o stargo main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/stargo .
COPY --from=builder /app/migrations ./migrations
CMD ["./stargo"]
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Go Chi](https://github.com/go-chi/chi) for the excellent HTTP router
- [pgx](https://github.com/jackc/pgx) for the powerful PostgreSQL driver
- [Zap](https://github.com/uber-go/zap) for structured logging
- [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations

---

**Stargo** - Build secure Go web APIs faster 🚀
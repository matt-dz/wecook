# WeCook Backend

RESTful API server for the WeCook recipe management application, built with Go and PostgreSQL.

## What It Does

The WeCook backend provides a complete API for managing recipes, users, and authentication. It handles:

- **User Authentication** - JWT-based auth with access and refresh tokens
- **Recipe Management** - CRUD operations for recipes with ingredients, steps, and images
- **Image Uploads** - File storage for recipe, ingredient, and step images
- **User Management** - Admin and user role-based access control
- **Recipe Publishing** - Public and private recipe visibility
- **Email Invitations** - SMTP-based user invitation system (optional)

## What It Contains

### Core Packages

- **`cmd/wecook`** - Application entry point
- **`internal/api`** - HTTP routes, handlers, and middleware
  - **`openapi/`** - Auto-generated OpenAPI models and server stubs
- **`internal/database`** - Database connection and SQLC-generated queries
- **`internal/sql`** - SQL schema and query definitions

### Feature Packages

- **`argon2id`** - Password hashing with Argon2id
- **`jwt`** - JWT token generation and validation
- **`password`** - Password strength validation
- **`role`** - User role management (admin, user)
- **`email`** - SMTP email sending for invitations
- **`fileserver`** - Static file serving
- **`filestore`** - File storage abstraction
- **`invite`** - User invitation system

### Utility Packages

- **`env`** - Environment variable configuration
- **`http`** - HTTP server setup and middleware
- **`log`** - Structured logging
- **`json`** - JSON encoding/decoding utilities
- **`file`** - File system utilities
- **`form`** - Form parsing helpers
- **`setup`** - Application initialization

### Documentation & Configuration

- **`docs/`** - OpenAPI 3.0 specification (`api.yaml`)
- **`Makefile`** - Build, test, and development commands
- **`sqlc.yaml`** - SQLC configuration for database code generation
- **`.env`** - Environment variables (not committed)

## Tech Stack

- **Go 1.25.1** - Primary language
- **Chi Router** - HTTP routing
- **PostgreSQL 18** - Database with pgx driver
- **OpenAPI 3.0** - API specification with oapi-codegen
- **SQLC** - Type-safe SQL queries
- **JWT** - Token-based authentication
- **Argon2id** - Password hashing
- **GoMock** - Mock generation for testing

## Prerequisites

- [Go 1.25.1+](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/engine/install/)
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/)
- [pg_format](https://github.com/darold/pgFormatter)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- [Make](https://www.gnu.org/software/make/)

Optional for hot reloading:
- [Air](https://github.com/air-verse/air)

## Getting Started

### 1. Install Dependencies

```bash
go mod download
```

### 2. Start Development Environment

The project uses Docker Compose for development, which automatically sets up PostgreSQL and all required services.

From the **project root directory** (not the backend directory):

```bash
# Copy environment files
cp .env.backend.example .env.backend
cp .env.database.example .env.database
cp .env.frontend.example .env.frontend

# Start all services (database, backend, frontend, nginx)
docker compose -f docker-compose.dev.yaml up -d
```

This will:
- Start PostgreSQL and apply the database schema
- Start backend with hot-reload enabled

### 3. Generate Code

```bash
# Generate OpenAPI models and server code
make docs

# Generate database query code
make sqlc

# Generate test mocks
make mocks
```

### 4. Verify Setup

```bash
# Check health
curl http://localhost:8080/api/ping

# Check logs
docker logs wecook_backend | tail -n 10

# View API documentation
open http://localhost:8080/api/docs
```

## Development Commands

```bash
make build      # Build the application
make run        # Run the server
make test       # Run all tests
make fmt        # Format Go code and SQL
make lint       # Run linter
make docs       # Generate OpenAPI code
make sqlc       # Generate database code
make mocks      # Generate test mocks
make clean      # Clean build artifacts
```

### Individual Mock Generation

```bash
make dbmock         # Database mocks
make fileservermock # File server mocks
make filestoremock  # File store mocks
make smtpmock       # SMTP mocks
```

## Project Architecture

### OpenAPI-First Development

The backend follows an **API-first** approach:

1. API is defined in `docs/api.yaml` (OpenAPI 3.0 spec)
2. Code is generated from the spec using `oapi-codegen`
3. Generated types and interfaces live in `internal/api/openapi/`
4. Handlers implement the generated server interface

### Database Layer

Database queries use **SQLC** for type safety:

1. Schema defined in `internal/sql/schema.sql`
2. Queries written in `internal/sql/query.sql`
3. SQLC generates type-safe Go code in `internal/database/`
4. No ORMs or query builders - just SQL and generated Go

### Request Flow

1. HTTP request → Chi router
2. OpenAPI middleware validates request
3. Handler extracts user from JWT
4. Handler calls database via SQLC-generated code
5. Response returned as JSON

### Testing Strategy

- **Unit Tests** - Table-driven tests with GoMock for database
- **Handler Tests** - Test API endpoints with mocked dependencies
- **No Integration Tests** - All tests run in-memory without real database

## Adding New Features

### 1. Add API Endpoint

Define in `docs/api.yaml`:

```yaml
paths:
  /api/your-endpoint:
    post:
      summary: Your endpoint
      description: Your description
      tags:
        - Tag
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/YourRequest"
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/YourResponse"
```

Generate code:

```bash
make docs
```

### 2. Add Database Query (if necessary)

If necessary, add relevant queries to `internal/sql/query.sql`:

```sql
-- name: GetYourData :many
SELECT id, name FROM your_table
WHERE user_id = $1;
```

Generate code:

```bash
make sqlc
```

### 3. Implement Handler

Find the generated function in `internal/api/openapi/client.gen.go` under the `StrictServerInterface` interface. 
Then, in `internal/api/openapi/yourfeature.go` implement the function:

```go
func (Server) PostApiYourEndpoint(ctx context.Context,
    request PostApiYourEndpointRequestObject) (
    PostApiYourEndpointResponseObject, error) {

    env := env.EnvFromCtx(ctx)
    userID, err := token.UserIDFromCtx(ctx)
    // ... implementation

    return PostApiYourEndpoint200JSONResponse{}, nil
}
```

### 4. Write Tests

In `internal/api/openapi/yourfeature_test.go`:

```go
func TestPostApiYourEndpoint(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := database.NewMockQuerier(ctrl)
    mockDB.EXPECT().GetYourData(gomock.Any(), gomock.Any()).Return(data, nil)

    // ... test implementation
}
```

Run tests:

```bash
make test
```

## Testing

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./internal/api/openapi/

# With verbose output
go test -v ./...

# With coverage
go test -cover ./...
```

### Writing Tests

Use table-driven patterns with mocks:

```go
tests := []struct {
    name       string
    setup      func(mockDB *database.MockQuerier)
    wantStatus int
    wantError  bool
}{
    {
        name: "successful creation",
        setup: func(mockDB *database.MockQuerier) {
            mockDB.EXPECT().CreateThing(gomock.Any(), gomock.Any()).Return(1, nil)
        },
        wantStatus: 201,
        wantError:  false,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        ctrl := gomock.NewController(t)
        defer ctrl.Finish()

        mockDB := database.NewMockQuerier(ctrl)
        tt.setup(mockDB)

        // ... run test
    })
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_SECRET` | JWT signing secret (auto-generated if empty) | - |
| `APP_SECRET_PATH` | Path to store generated secret | `/data/secret` |
| `HOST_ORIGIN` | Application host URL | - |
| `DATABASE_USER` | PostgreSQL username | - |
| `DATABASE_PASSWORD` | PostgreSQL password | - |
| `DATABASE_HOST` | PostgreSQL host | `localhost` |
| `DATABASE_PORT` | PostgreSQL port | `5432` |
| `DATABASE` | Database name | - |
| `FILESERVER_VOLUME` | Path for uploaded files | `/data/files` |
| `ADMIN_EMAIL` | Initial admin email | - |
| `ADMIN_PASSWORD` | Initial admin password (min 10 chars, requires number, special char, upper/lowercase) | - |
| `SMTP_HOST` | SMTP server (optional) | - |
| `SMTP_PORT` | SMTP port (optional) | `587` |
| `SMTP_USERNAME` | SMTP username (optional) | - |
| `SMTP_PASSWORD` | SMTP password (optional) | - |
| `SMTP_FROM` | SMTP from address (optional) | - |
| `SMTP_TLS_MODE` | TLS mode (`auto`, `starttls`, `implicit`, `none`) | `auto` |
| `SMTP_TLS_SKIP_VERIFY` | Skip TLS certificate verification (development only) | `false` |

With `SMTP_TLS_MODE=auto`, port `587` uses `STARTTLS` after `EHLO`, port `465` uses implicit TLS, and other ports send without TLS.

## API Documentation

### Accessing Docs

When running locally:
- **Swagger UI**: http://localhost:8080/api/docs
- **OpenAPI Spec**: http://localhost:8080/api/openapi.yaml

## Security

- **Passwords** - Hashed with Argon2id (memory-hard, resistant to GPU cracking)
- **JWTs** - Signed with HS256, 30-minute expiry for access tokens
- **Refresh Tokens** - 14-day expiry, stored as hashed values in database
- **SQL Injection** - Prevented via SQLC parameterized queries
- **Input Validation** - Automatic validation via OpenAPI middleware
- **File Uploads** - Content-type validation and size limits

## Contributing

See the [main README](../README.md) for general contribution guidelines.

### Backend-Specific Guidelines

1. **OpenAPI First** - Always update `docs/api.yaml` before implementing endpoints
2. **No ORMs** - Use SQLC for database queries
3. **Table-Driven Tests** - Follow existing test patterns
4. **Mock Dependencies** - Use GoMock for all external dependencies
5. **Error Handling** - Only validate at system boundaries
6. **No Over-Engineering** - Keep solutions simple and focused

## Additional Resources

- [OpenAPI Specification](https://swagger.io/specification/)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Chi Router](https://go-chi.io/)
- [GoMock](https://github.com/uber-go/mock)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

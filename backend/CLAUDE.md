# WeCook Backend - Claude Code Guide

This document provides context for Claude Code when working on the WeCook backend application.

## Project Overview

WeCook is a recipe management application with a Go backend API and SvelteKit frontend. The backend is a RESTful API that handles user authentication, recipe management, and file serving.

## Tech Stack

- **Language**: Go 1.25.1
- **Web Framework**: Chi router (v5)
- **Database**: PostgreSQL with pgx driver
- **Auth**: JWT tokens with golang-jwt
- **Password Hashing**: Argon2id
- **API Documentation**: OpenAPI 3.0 with oapi-codegen
- **Code Generation**: oapi-codegen for type-safe API models and server stubs
- **Database Queries**: SQLC for type-safe SQL
- **Development**: Air for hot reloading

## Project Structure

```
backend/
├── cmd/wecook/          # Application entry point
├── internal/            # Internal packages
│   ├── api/            # API routes and handlers
│   ├── database/       # Database connection and queries
│   ├── dbmock/         # Generated database mocks for testing
│   ├── sql/            # SQL schema and queries
│   ├── jwt/            # JWT token handling
│   ├── password/       # Password validation
│   ├── argon2id/       # Password hashing
│   ├── role/           # User role management
│   ├── recipe/         # Recipe-related logic
│   ├── fileserver/     # Static file serving
│   ├── http/           # HTTP utilities
│   ├── json/           # JSON encoding/decoding
│   ├── env/            # Environment variable handling
│   └── log/            # Logging utilities
├── docs/               # Swagger documentation
├── bin/                # Compiled binaries
└── keys/               # JWT signing keys
```

## Key Dependencies

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/httplog/v3` - HTTP logging middleware
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/golang-jwt/jwt/v5` - JWT implementation
- `github.com/oapi-codegen/oapi-codegen` - OpenAPI code generator
- `github.com/oklog/ulid/v2` - ULID generation
- `golang.org/x/crypto` - Argon2 password hashing
- `go.uber.org/mock/gomock` - Mock generation for testing

## Database Schema

### Core Tables

- **users**: User accounts with email, name, role, password hash, and refresh tokens
- **recipes**: Recipe data including title, description, images, cook/prep time, servings
- **recipe_ingredients**: Ingredients for recipes with quantities and units
- Additional tables for recipe steps and other features

### Custom Types

- `ROLE` enum: 'admin', 'user'
- `time_unit` enum: 'minutes', 'hours', 'days'

## Development Workflow

### Common Commands

```bash
make build      # Build the application (fmt, lint, docs, compile)
make run        # Run the application with code generation
make test       # Run all tests
make fmt        # Format Go code and SQL
make lint       # Run linter
make docs       # Generate OpenAPI client, models, and server code
make sqlc       # Generate SQLC code from SQL files
make dbmock     # Generate database mocks for testing
make keys       # Generate JWT signing keys
```

### Environment Variables

See `.env.example` for required environment variables. Key variables include:
- Database connection settings
- JWT configuration
- Server port and settings

### Docker

- `docker-compose.yaml` - Orchestrates backend and database
- `Dockerfile` - Production build
- `Dockerfile.dev` - Development build with hot reloading

## Code Style and Patterns

### General Principles

1. **No Over-Engineering**: Keep solutions simple and focused on the requested task
2. **Minimal Abstractions**: Don't create utilities or helpers for one-time operations
3. **Error Handling**: Only validate at system boundaries (user input, external APIs)
4. **Security**: Watch for OWASP Top 10 vulnerabilities (SQL injection, XSS, etc.)

### Project Conventions

1. **Internal Packages**: All application code lives in `internal/` to prevent external imports
2. **SQLC**: Database queries are written in SQL and type-safe Go code is generated
3. **OpenAPI-First**: API is defined in `docs/api.yaml` and code is generated from the specification
4. **Request Validation**: oapi-codegen middleware validates requests against OpenAPI spec
5. **Logging**: Use httplog for structured HTTP logging

## Testing

### Overview

The project uses Go's built-in testing framework with a focus on unit testing business logic and API handlers.

### Test Files

Tests are located alongside the code they test with the `_test.go` suffix:
- `internal/api/openapi/admin_test.go` - API handler tests
- `internal/fileserver/fileserver_test.go` - File server tests

### Running Tests

```bash
make test       # Run all tests
go test ./...   # Run tests directly
```

### Database Mocking

The project uses **GoMock** to generate mocks of the database interface for testing without a real database.

#### Mock Generation

Database mocks are generated from the SQLC-generated `Querier` interface:

```bash
make dbmock     # Generate database mocks
```

This runs:
```bash
mockgen -source=internal/database/querier.go \
    -destination internal/dbmock/dbmock.go \
    -package dbmoc
```

- **Source**: `internal/database/querier.go` - SQLC-generated interface
- **Destination**: `internal/dbmock/dbmock.go` - Generated mock
- **Package**: `dbmoc` - Mock package name

#### Using Mocks in Tests

1. **Import the mock package**:
   ```go
   import (
       dbmoc "github.com/matt-dz/wecook/internal/dbmock"
       "go.uber.org/mock/gomock"
   )
   ```

2. **Create a mock controller**:
   ```go
   ctrl := gomock.NewController(t)
   defer ctrl.Finish()
   mockDB := dbmoc.NewMockQuerier(ctrl)
   ```

3. **Set expectations**:
   ```go
   mockDB.EXPECT().
       CreateAdmin(gomock.Any(), gomock.Any()).
       Return(int64(1), nil)
   ```

4. **Inject mock into context**:
   ```go
   ctx := env.WithCtx(ctx, &env.Env{
       Logger: log.NullLogger(),
       Database: &database.Database{
           Querier: mockDB,
       },
   })
   ```

### Testing Patterns

#### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
tests := []struct {
    name       string
    request    PostApiAdminRequestObject
    setup      func()
    wantStatus int
    wantCode   string
    wantError  bool
}{
    {
        name: "successful creation",
        request: PostApiAdminRequestObject{...},
        setup: func() {
            mockDB.EXPECT().CreateAdmin(...).Return(1, nil)
        },
        wantStatus: 204,
    },
    // More test cases...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        tt.setup()
        // Run test...
    })
}
```

#### API Handler Tests

When testing API handlers:

1. Create a mock database
2. Set up expectations for database calls
3. Create a context with request ID and environment
4. Call the handler function
5. Assert response type and values

Example:
```go
func TestPostApiAdmin(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := dbmoc.NewMockQuerier(ctrl)
    mockDB.EXPECT().CreateAdmin(gomock.Any(), gomock.Any()).Return(int64(1), nil)

    ctx := requestid.InjectRequestID(context.Background(), 12345)
    ctx = env.WithCtx(ctx, &env.Env{
        Logger: log.NullLogger(),
        Database: &database.Database{Querier: mockDB},
    })

    server := NewServer()
    resp, err := server.PostApiAdmin(ctx, request)

    // Assert response...
}
```

#### Testing Error Cases

Test various error scenarios:
- Weak passwords
- Duplicate emails (PostgreSQL unique constraint violations)
- Database connection errors
- Invalid input

Use PostgreSQL error codes for constraint violations:
```go
pgErr := &pgconn.PgError{
    Code:           "23505",  // unique_violation
    ConstraintName: "users_unique_email",
}
mockDB.EXPECT().CreateAdmin(...).Return(int64(0), pgErr)
```

#### Helper Functions

Create test helpers to reduce boilerplate:
```go
func newTestFileServer(t *testing.T) (*FileServer, string) {
    t.Helper()
    base := t.TempDir()
    return &FileServer{baseDir: base}, base
}
```

### Testing Best Practices

1. **Parallel Tests**: Use `t.Parallel()` for independent tests
2. **Cleanup**: Use `defer ctrl.Finish()` for mock controllers
3. **Temp Directories**: Use `t.TempDir()` for file system tests
4. **Test Isolation**: Each test should be independent
5. **Clear Names**: Use descriptive test case names
6. **Null Logger**: Use `log.NullLogger()` to suppress log output in tests

## Security Considerations

- **Passwords**: Hashed with Argon2id before storage
- **Authentication**: JWT-based with refresh tokens
- **Authorization**: Role-based access control (admin, user)
- **Database**: Parameterized queries via SQLC prevent SQL injection
- **Input Validation**: Validator package for request validation

## API Documentation

The project uses an **OpenAPI-first** approach where the API specification is the source of truth:

### OpenAPI Specification

- Location: `docs/api.yaml`
- Version: OpenAPI 3.0.3
- Defines all endpoints, request/response schemas, and security requirements

### Code Generation

The project uses `oapi-codegen` to generate type-safe Go code from the OpenAPI specification:

- **Configuration**: `docs/cfg.yaml`
- **Generated code**: `internal/api/openapi/client.gen.go`
- **Generates**:
  - Request/response models (type-safe structs)
  - Chi server interface and router stubs
  - Client code for API consumption
  - Strict server handlers

### Workflow

1. Define or update endpoints in `docs/api.yaml`
2. Run `make docs` to generate Go code
3. Implement handler functions matching the generated interfaces
4. Register routes in `internal/api/api.go`

### Request Validation

- Automatic validation via `oapi-codegen/nethttp-middleware`
- Validates request bodies, query params, and path params against OpenAPI spec
- Returns 400 Bad Request for invalid requests

## Working with the Frontend

The frontend is a SvelteKit application located in `../frontend/`. When making backend changes that affect the API:

1. Update API handlers and Swagger documentation
2. Ensure response formats match frontend expectations
3. Test with the frontend if making breaking changes

## Recent Changes

Based on git history:
- Implemented basic recipe editing page
- Split cook_time_minutes into amount and unit fields
- Added prep_time fields
- Added servings field to recipes

## Common Tasks

### Adding a New API Endpoint

The project follows an OpenAPI-first workflow. When adding a new endpoint:

1. **Define the endpoint in OpenAPI spec** (`docs/api.yaml`):
   ```yaml
   paths:
     /api/your-endpoint:
       post:
         summary: Your endpoint description
         tags: [YourTag]
         requestBody:
           required: true
           content:
             application/json:
               schema:
                 $ref: "#/components/schemas/YourRequestSchema"
         responses:
           "200":
             description: Success
             content:
               application/json:
                 schema:
                   $ref: "#/components/schemas/YourResponseSchema"
   ```

2. **Define schemas in components section**:
   ```yaml
   components:
     schemas:
       YourRequestSchema:
         type: object
         required: [field1]
         properties:
           field1:
             type: string
   ```

3. **Generate code**: Run `make docs` to generate models and server stubs

4. **Implement the handler** in `internal/api/routes/yourfeature/`:
   - Use generated request/response types from `internal/api/openapi/client.gen.go`
   - Handle business logic
   - Return appropriate status codes

5. **Register the route** in `internal/api/api.go`:
   ```go
   r.Post("/your-endpoint", yourfeature.HandleYourEndpoint)
   ```

6. **Add database queries** if needed:
   - Update `internal/sql/query.sql`
   - Run `make sqlc` to generate Go code

7. **Test the endpoint**: The oapi-codegen middleware will automatically validate requests

### Database Schema Changes

1. Update `internal/sql/schema.sql`
2. Run `make sql-fmt` to format SQL
3. Update queries in `internal/sql/query.sql` as needed
4. Run `make sqlc` to regenerate Go code
5. Create migration strategy for production

### Adding Validation

Validation is handled automatically by the OpenAPI middleware:

1. **Schema-level validation**: Define constraints in `docs/api.yaml`:
   ```yaml
   properties:
     age:
       type: integer
       minimum: 0
       maximum: 150
     email:
       type: string
       format: email
   ```

2. **Required fields**: Mark fields as required in the schema:
   ```yaml
   required: [field1, field2]
   ```

3. **Request validation**: The `oapimw.OapiRequestValidatorWithOptions` middleware automatically validates all requests against the OpenAPI spec

4. **Custom validation**: For business logic validation beyond OpenAPI constraints, implement checks in your handlers

## Notes for Claude Code

- Always read existing code before making changes
- Use `make build` to verify changes compile and pass linting
- **IMPORTANT**: Update `docs/api.yaml` FIRST when adding/modifying API endpoints
- Run `make docs` after OpenAPI spec changes to regenerate code
- Run `make sqlc` after SQL changes
- Run `make dbmock` after modifying the `Querier` interface (happens automatically with SQLC changes)
- Keep security in mind (especially auth, password handling, SQL)
- Follow existing patterns in the codebase
- Don't add unnecessary comments or documentation
- Test changes by running `make run` and checking the API
- Run `make test` to verify all tests pass
- Write tests for new features using table-driven patterns with database mocks
- The OpenAPI spec is the source of truth for the API - always keep it in sync with implementation

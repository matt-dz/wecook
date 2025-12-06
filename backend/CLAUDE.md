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
- **API Documentation**: Swagger/OpenAPI
- **Database Queries**: SQLC for type-safe SQL
- **Development**: Air for hot reloading

## Project Structure

```
backend/
├── cmd/wecook/          # Application entry point
├── internal/            # Internal packages
│   ├── api/            # API routes and handlers
│   ├── database/       # Database connection and queries
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
- `github.com/swaggo/swag` - Swagger documentation generator
- `github.com/oklog/ulid/v2` - ULID generation
- `golang.org/x/crypto` - Argon2 password hashing

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
make build      # Build the application (fmt, lint, swagger, compile)
make run        # Run the application with swagger generation
make fmt        # Format Go code, swagger docs, and SQL
make lint       # Run linter
make swagger    # Generate swagger documentation
make sqlc       # Generate SQLC code from SQL files
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
3. **Swagger**: API documentation is generated from code comments using swag
4. **Validation**: Use go-playground/validator for input validation
5. **Logging**: Use httplog for structured HTTP logging

## Testing

- Focus on testing business logic in internal packages
- Use table-driven tests where appropriate
- Mock database interactions when needed

## Security Considerations

- **Passwords**: Hashed with Argon2id before storage
- **Authentication**: JWT-based with refresh tokens
- **Authorization**: Role-based access control (admin, user)
- **Database**: Parameterized queries via SQLC prevent SQL injection
- **Input Validation**: Validator package for request validation

## API Documentation

- Swagger docs available at `/swagger/index.html` when running
- Generated from code comments using swag
- Run `make swagger` to regenerate after API changes

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

1. Add handler function in `internal/api/`
2. Add route in API router
3. Add Swagger comments above handler
4. Run `make swagger` to update docs
5. Add SQL queries if needed in `internal/sql/query.sql`
6. Run `make sqlc` to generate Go code

### Database Schema Changes

1. Update `internal/sql/schema.sql`
2. Run `make sql-fmt` to format SQL
3. Update queries in `internal/sql/query.sql` as needed
4. Run `make sqlc` to regenerate Go code
5. Create migration strategy for production

### Adding Validation

1. Add validator tags to struct fields
2. Use validator instance in handlers
3. Return appropriate error responses

## Notes for Claude Code

- Always read existing code before making changes
- Use `make build` to verify changes compile and pass linting
- Update Swagger docs when modifying API endpoints
- Run `make sqlc` after SQL changes
- Keep security in mind (especially auth, password handling, SQL)
- Follow existing patterns in the codebase
- Don't add unnecessary comments or documentation
- Test changes by running `make run` and checking the API

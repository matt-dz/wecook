# WeCook

A self-hosted recipe manager for organizing and sharing your favorite recipes.

<img width="1466" height="832" alt="Screenshot 2025-12-28 at 3 47 20 PM" src="https://github.com/user-attachments/assets/a0f1eb94-ad8e-46ad-b56e-c9c939101b9d" />

<img width="1464" height="833" alt="Screenshot 2025-12-28 at 3 48 59 PM" src="https://github.com/user-attachments/assets/be497486-20b1-40df-bbaa-59749eb89c25" />

<img width="1466" height="831" alt="Screenshot 2025-12-28 at 3 49 40 PM" src="https://github.com/user-attachments/assets/abb688c7-e7a0-4046-a5e5-673b1dc9c1f0" />


## Features

- **Recipe Management** - Create, edit, and organize recipes with ingredients, steps, and images
- **Recipe Publishing** - Share recipes publicly or keep them private
- **RESTful API** - OpenAPI-documented REST API for all operations

## Project Structure

```
wecook/
├── backend/              # Go API server
│   ├── cmd/wecook/      # Application entry point
│   ├── internal/        # Internal packages
│   │   ├── api/        # API routes and handlers
│   │   ├── database/   # Database connection and queries
│   │   ├── sql/        # SQL schema and queries
│   │   └── ...         # Other internal packages
│   ├── docs/           # OpenAPI specification
│   └── Makefile        # Build commands
├── frontend/            # SvelteKit web application
│   ├── src/            # Source code
│   ├── static/         # Static assets
│   └── package.json    # Node dependencies
├── docker-compose.yaml  # Production orchestration
├── docker-compose.dev.yaml # Development orchestration
└── fileserver.conf     # Nginx configuration

```

## Prerequisites

### For Docker Deployment (Recommended)
- [Docker](https://docs.docker.com/engine/install/)

### For Local Development
- [Go 1.25.1+](https://go.dev/doc/install)
- [Node.js 20+](https://nodejs.org/en/download)
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/)
- [pg_format (PostgreSQL utilities)](https://github.com/darold/pgFormatter)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- [Make](https://www.gnu.org/software/make/)

## Quick Start with Docker

### 1. Download Files

```bash
# Docker Compose
wget https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/docker-compose.yaml

# NGINX Config
wget https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/fileserver.conf

# .env files
wget -O .env.backend https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/.env.backend.example
wget -O .env.frontend https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/.env.frontend.example
wget -O .env.database https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/.env.database.example
```

### 2. Configure Environment Variables

Edit `.env.backend` and set:
- (recommended) change `DATABASE_PASSWORD` to a secure password. You may generate a password with the following command `openssl rand --base64 32`.
    - **Note**: Ensure the database variables correspond to their respective values in `.env.database`.
- `ADMIN_EMAIL` and `ADMIN_PASSWORD` - You will use these to login.
- (optional) `HOST_ORIGIN` (e.g., `http://localhost:8080`)
- (optional) Email configurartion - this will be used to invite users.

Edit `.env.database` and set:
- PostgreSQL credentials

### 3. Start the Application

```bash
docker compose up -d
```

The application will be available at `http://localhost:8080`

### 4. Access the Application

- **Web Interface**: http://localhost:8080
- **API**: http://localhost:8080/api
- **API Documentation**: http://localhost:8080/api/docs

Default admin credentials are set in `.env.backend`:
- Email: `admin@example.com`
- Password: `Change-m3!` (change this!)


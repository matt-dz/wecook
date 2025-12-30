# WeCook

A self-hosted recipe manager for organizing and sharing your favorite recipes. Checkout the demo website [wecook.deguzman.cloud](https://wecook.deguzman.cloud).

<img width="1466" height="832" alt="Screenshot 2025-12-28 at 3 47 20 PM" src="https://github.com/user-attachments/assets/a0f1eb94-ad8e-46ad-b56e-c9c939101b9d" />

<img width="1464" height="833" alt="Screenshot 2025-12-28 at 3 48 59 PM" src="https://github.com/user-attachments/assets/be497486-20b1-40df-bbaa-59749eb89c25" />

<img width="1466" height="831" alt="Screenshot 2025-12-28 at 3 49 40 PM" src="https://github.com/user-attachments/assets/abb688c7-e7a0-4046-a5e5-673b1dc9c1f0" />

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Quick Start with Docker](#quick-start-with-docker)
- [Admin Dashboard](#admin-dashboard)
  - [User Management](#user-management)
  - [Application Preferences](#application-preferences)
- [Configuration](#configuration)
  - [Backend Environment Variables](#backend-environment-variables)
  - [Database Environment Variables](#database-environment-variables)
  - [Frontend Environment Variables](#frontend-environment-variables)
- [Kubernetes Deployment](#kubernetes-deployment)
- [License](#license)

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

### 2. Configure the Application

WeCook supports two configuration methods:
1. **YAML Configuration File** (recommended)
2. **Environment Variables** (alternative method)

**Note**: The application will run with default values. You only need to configure for production use.

#### Option A: YAML Configuration (Recommended)

Create a `wecook.yaml` file and mount it to `/data/wecook.yaml` in the container.

Download the example configuration:
```bash
wget https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/wecook.yaml
```

Edit `wecook.yaml` with your settings. All settings are documented with comments in the file.

Mount the file in your `docker-compose.yaml`:
```yaml
services:
  wecook-backend:
    volumes:
      - ./wecook.yaml:/data/wecook.yaml:ro
```

See the [YAML Configuration Reference](#yaml-configuration-reference) section below for all available options.

#### Option B: Environment Variables

#### Required for Production

Edit both `.env.backend` and `.env.database`:

1. **Database Password** - Set the same password in both files:
   - In `.env.backend`: `DATABASE_PASSWORD=your-secure-password`
   - In `.env.database`: `POSTGRES_PASSWORD=your-secure-password`
   - Generate a secure password: `openssl rand --base64 32`

2. **Admin Credentials** - Set in `.env.backend`:
   - `ADMIN_EMAIL=your-email@example.com`
   - `ADMIN_PASSWORD=Your-Secure-Password1!`

#### Optional Configuration

Edit `.env.backend` if needed:

- **`HOST_ORIGIN`** - Only needed if serving at a different URL (e.g., `https://example.com`)
  - Default: `http://localhost:8080`
- **SMTP Settings** - Only needed for user invitations via email:
  - `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`

### 3. Start the Application

```bash
docker compose up -d
```

The application will be available at `http://localhost:8080`

### 4. Access the Application

- **Web Interface**: http://localhost:8080
- **API**: http://localhost:8080/api
- **API Documentation**: http://localhost:8080/docs/

Login with the admin credentials you set in `.env.backend` (default: `admin@example.com` / `Change-m3!`)

## Admin Dashboard

The admin dashboard provides tools for managing users and configuring application settings. Access it by logging in with an admin account and navigating to `/admin`.

### User Management

<img width="1470" height="835" alt="Screenshot 2025-12-30 at 5 28 35 PM" src="https://github.com/user-attachments/assets/df913b7f-4179-4fc1-9830-e995f671e0e8" />

As an admin, you have full control over user accounts:

- **View All Users** - See a list of all registered users with their email addresses, names, and roles
- **Invite Users** - Send invitation emails to new users
  - When SMTP is configured, invitation emails are sent automatically with a signup link
  - Invitations expire after a configurable period
  - Without SMTP configuration, you can still generate invite codes to share manually
- **Delete Users** (TODO) - Remove user accounts from the system
  - Deleting a user removes their account and all associated data
  - This action cannot be undone

### Application Preferences

<img width="1470" height="835" alt="Screenshot 2025-12-30 at 5 29 11 PM" src="https://github.com/user-attachments/assets/91c123ff-5184-45a9-a159-addc88ded354" />

Control how WeCook operates for all users:

- **Allow Public Signup**
  - When enabled, anyone can create an account without an invitation code
  - When disabled, new users must have a valid invitation code to sign up
  - Useful for controlling access to your WeCook instance

## Configuration

WeCook can be configured using either a YAML configuration file (recommended) or environment variables. If a YAML file is present at `/data/wecook.yaml`, it will be used. Otherwise, the application will load configuration from environment variables.

### YAML Configuration Reference

Mount your `wecook.yaml` file to `/data/wecook.yaml` in the container. See the example [wecook.yaml](wecook.yaml) file in the repository for a complete, commented configuration template.

**Example minimal configuration:**
```yaml
env: PROD
host_origin: https://your-domain.com

database:
  host: postgres
  port: 5432
  database: wecook
  user: wecook
  password: your-secure-password

admin:
  first_name: Admin
  last_name: User
  email: admin@your-domain.com
  password: Your-Secure-Password1!
```

For complete documentation of all configuration options, see [wecook.yaml](wecook.yaml).

### Backend Environment Variables

Configuration file: `.env.backend`

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_SECRET` | JWT signing secret (auto-generated if not set, must be at least 32 bytes if provided) | Auto-generated | No |
| `APP_SECRET_PATH` | Path to store auto-generated secret | `/data/secret` | No |
| `APP_SECRET_VERSION` | Version identifier for JWT secret (for key rotation) | `1` | No |
| `ENV` | Environment mode (`PROD` for production, anything else for development) | Development | No |
| `HOST_ORIGIN` | Application host URL for CORS and cookies | `http://localhost:8080` | Yes |
| `DATABASE_USER` | PostgreSQL username | - | Yes |
| `DATABASE_PASSWORD` | PostgreSQL password | - | Yes |
| `DATABASE_HOST` | PostgreSQL hostname | `localhost` | Yes |
| `DATABASE_PORT` | PostgreSQL port | `5432` | Yes |
| `DATABASE` | PostgreSQL database name | - | Yes |
| `FILESERVER_VOLUME` | Path for uploaded files | `/data/files` | Yes |
| `FILESERVER_URL_PREFIX` | URL prefix for served files | `/files` | No |
| `ADMIN_FIRST_NAME` | Initial admin user first name | - | No* |
| `ADMIN_LAST_NAME` | Initial admin user last name | - | No* |
| `ADMIN_EMAIL` | Initial admin user email | - | No* |
| `ADMIN_PASSWORD` | Initial admin password (min 10 chars, requires number, special char, upper/lowercase) | - | No* |
| `SMTP_HOST` | SMTP server hostname for email invitations | - | No** |
| `SMTP_PORT` | SMTP server port | `587` | No** |
| `SMTP_USERNAME` | SMTP authentication username | - | No** |
| `SMTP_PASSWORD` | SMTP authentication password | - | No** |
| `SMTP_FROM` | Email sender address | - | No** |
| `SMTP_TLS_MODE` | TLS mode: `auto`, `starttls`, `implicit`, or `none` | `auto` | No** |
| `SMTP_TLS_SKIP_VERIFY` | Skip TLS certificate verification (development only) | `false` | No** |

\* Required if you want to create an admin user on first startup

\** Required only if you want email invitation functionality

**Notes:**
- With `SMTP_TLS_MODE=auto`: port 587 uses STARTTLS, port 465 uses implicit TLS, other ports send without TLS
- Admin credentials are only used on first startup when no admin exists
- `APP_SECRET` is automatically generated and persisted if not provided

### Database Environment Variables

Configuration file: `.env.database`

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POSTGRES_USER` | PostgreSQL database username | - | Yes |
| `POSTGRES_PASSWORD` | PostgreSQL database password | - | Yes |
| `POSTGRES_DB` | PostgreSQL database name | - | Yes |

**Important:** `POSTGRES_PASSWORD` must match `DATABASE_PASSWORD` in `.env.backend`

### Frontend Environment Variables

Configuration file: `.env.frontend`

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `INTERNAL_BACKEND_URL` | Internal backend API URL for server-side requests | `http://wecook-backend:8080` | Yes |
| `CHOKIDAR_USEPOLLING` | Enable polling for file watching (development only) | `true` | No |
| `WATCHPACK_POLLING` | Enable polling for webpack watching (development only) | `true` | No |

**Notes:**
- `INTERNAL_BACKEND_URL` is used for server-side API calls within the Docker network
- Polling variables are only needed for development with Docker on certain filesystems

## Kubernetes Deployment

Kubernetes manifests that mirror the Docker Compose stack are available in [`k8s/`](k8s/). See [`k8s/README.md`](k8s/README.md) for configuration notes.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

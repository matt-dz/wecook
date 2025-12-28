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

**Note**: The application will run with default values. You only need to configure variables for production use.

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

## Kubernetes Deployment

Kubernetes manifests that mirror the Docker Compose stack are available in [`k8s/`](k8s/). Review and update the bundled secrets before applying them:

```bash
kubectl apply -f k8s/storage.yaml
kubectl apply -f k8s/database.yaml
kubectl apply -f k8s/backend.yaml
kubectl apply -f k8s/frontend.yaml
kubectl apply -f k8s/swagger-ui.yaml
kubectl apply -f k8s/nginx.yaml
```

See [`k8s/README.md`](k8s/README.md) for configuration notes.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

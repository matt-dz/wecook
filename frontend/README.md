# WeCook Frontend

SvelteKit web application for the WeCook recipe management platform.

## Backend

For backend development and API documentation, see the [Backend README](../backend/README.md).

## Developing

### Recommended: Using Docker Compose

The recommended way to develop the frontend is using Docker Compose, which runs the frontend with hot-reload alongside the backend and database.

From the **project root directory**:

```bash
# Copy environment files (first time only)
cp .env.backend.example .env.backend
cp .env.database.example .env.database
cp .env.frontend.example .env.frontend

# Start all services (database, backend, frontend, nginx)
docker compose -f docker-compose.dev.yaml up -d

# View logs
docker compose -f docker-compose.dev.yaml logs -f web

# Restart frontend only
docker compose -f docker-compose.dev.yaml restart web
```

The application will be available at `http://localhost:8080`

Changes to the frontend source code will automatically trigger a rebuild.

### Alternative: Local Development

If you need to run the frontend locally outside of Docker:

Install dependencies:

```bash
npm install
```

Start the development server:

```bash
npm run dev

# or start and open in browser
npm run dev -- --open
```

## Building

Create a production build:

```bash
npm run build
```

Preview the production build:

```bash
npm run preview
```

## Additional Commands

```bash
npm run check       # Type-check
npm run lint        # Lint code
npm run format      # Format code
```

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

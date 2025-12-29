# Kubernetes deployment manifests

This directory provides Kubernetes manifests mirroring the existing Docker Compose setup:

- PostgreSQL database (`k8s/database.yaml`)
- Backend API (`k8s/backend.yaml`)
- Frontend (`k8s/frontend.yaml`)
- Swagger UI docs (`k8s/swagger-ui.yaml`)
- Nginx fileserver for uploads (`k8s/fileserver.yaml`)
- Cluster Ingress for routing to services (`k8s/ingress.yaml`)
- Shared storage claims for uploads and backend data (`k8s/storage.yaml`)

## Before applying

1. Update the secrets in `k8s/backend.yaml` and `k8s/database.yaml` with production-grade values:
   - `wecook-app-secret` for `APP_SECRET`, `ADMIN_PASSWORD`, and `SMTP_PASSWORD`.
   - `wecook-db-secret` for `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.
   - **Note**: Generate secure passwords with openssl: `openssl rand --base64 32`.
2. Update `HOST_ORIGIN` in `k8s/backend.yaml` to the origin the app will be served at (i.e. wecook.deguzman.cloud).
3. Ensure an Ingress controller (for example, `ingress-nginx`) is installed in your cluster before applying `k8s/ingress.yaml`.

## Apply order

1. Deploy the PVCs, fileserver, frontend, database. Wait for the database to be ready before continuing.

```bash
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/storage.yaml
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/fileserver.yaml
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/frontend.yaml
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/database.yaml
kubectl wait --for=condition=Ready pods -l app=wecook-database
```

2. Deploy and wait for the backend.

```bash
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/backend.yaml
kubectl wait --for=condition=Ready pods -l app=wecook-backend
```

3. Deploy swagger-ui and the ingress.

```bash
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/swagger-ui.yaml
kubectl apply -f https://raw.githubusercontent.com/matt-dz/wecook/refs/heads/main/k8s/ingress.yaml
```

The Ingress routes `/api` to the backend, `/files` to the dedicated nginx fileserver, `/docs` to Swagger UI, and `/` to the frontend.

4. (optional) If using minikube, run

```bash
minikube tunnel
```

The app will be available at http://127.0.0.1.

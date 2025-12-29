# Kubernetes deployment manifests

This directory provides Kubernetes manifests mirroring the existing Docker Compose setup:

- PostgreSQL database (`k8s/database.yaml`)
- Backend API (`k8s/backend.yaml`)
- Frontend (`k8s/frontend.yaml`)
- Swagger UI docs (`k8s/swagger-ui.yaml`)
- Cluster Ingress for routing to services (`k8s/ingress.yaml`)
- Shared storage claims for uploads and backend data (`k8s/storage.yaml`)

## Before applying

1. Update the secrets in `k8s/backend.yaml` and `k8s/database.yaml` with production-grade values:
   - `wecook-app-secret` for `APP_SECRET`, `ADMIN_PASSWORD`, and `SMTP_PASSWORD`.
   - `wecook-db-secret` for `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.
2. Confirm the storage class supports the requested access modes. `wecook-files` is defined as `ReadWriteMany` so nginx and the backend can both read the uploaded files.
3. Ensure an Ingress controller (for example, `ingress-nginx`) is installed in your cluster before applying `k8s/ingress.yaml`.

## Apply order

```bash
kubectl apply -f k8s/storage.yaml
kubectl apply -f k8s/database.yaml
kubectl apply -f k8s/backend.yaml
kubectl apply -f k8s/frontend.yaml
kubectl apply -f k8s/swagger-ui.yaml
kubectl apply -f k8s/ingress.yaml
```

The Ingress routes `/api` and `/files` to the backend, `/docs` to Swagger UI, and `/` to the frontend.

# ERP System

Monorepo ERP project with a Go backend, React frontend, PostgreSQL database, and Docker-based development environment.

## Getting Started

1. Copy `.env.example` to `.env` and update the required configuration.
2. Start the development environment:
   ```bash
   docker compose up -d
   ```
3. Check the running services:
   ```bash
   docker compose ps
   ```

## Development Environment

The project uses Docker Compose for local development: the backend runs with Go + Air for hot reload, the frontend runs with Vite HMR, and PostgreSQL uses persistent volume storage.

Normal source-code changes don't require rebuilding images. A rebuild is only needed when `Dockerfile`/`Dockerfile.dev`, `go.mod`/`go.sum`, `package.json`/`package-lock.json`, or `docker-compose.yml` change.

## Project Structure

```text
ERP-system/
├── backend/              # Go REST API + WebSocket
├── frontend/              # React + TypeScript + Vite
├── docker-compose.yml     # Development environment
└── .env.example           # Environment configuration template
```

## Features

### Authentication & Authorization
JWT-based auth with access and refresh tokens. Refresh tokens are stored opaque and SHA-256 hashed, with rotation, family tracking, and automatic family revocation on reuse detection. Authorization is permission-based RBAC, tied to user roles.

### User & Access Management
CRUD for users, roles, and permissions, including role-permission and user-role relationships.

### Audit Logging
Centralized audit log tracking actor, action, resource, resource ID, timestamp, and optional metadata. Covers authentication, product, and refresh token events.

### Product Master Data
Product CRUD with unique SKU (and optional barcode) validation, name validation, non-negative price validation (purchase & selling), soft delete, active/inactive state, and audit trail.
RBAC permissions: `products.read`, `products.create`, `products.update`, `products.delete`.

### Infrastructure & Development
Go backend, React + TypeScript frontend, PostgreSQL, Docker Compose for development. Production builds use multi-stage Docker images — a minimal `scratch` runtime for the backend and Nginx for the frontend. Backend hot reload via Air, frontend HMR via Vite, with a persistent Postgres volume and healthcheck.

## Architecture

The project is developed incrementally toward a modular ERP architecture:

```text
Authentication → Users → Roles → Permissions → RBAC → Audit Log
→ Product Master Data → Inventory → Sales → Purchasing
→ Finance → Reporting → Realtime → HR
```

Only implemented features are considered production-ready. Remaining modules are planned and developed incrementally.

## Testing

```bash
cd backend
go test ./... -v          # run tests
go test ./... -race       # with race detection
```

```bash
cd frontend
npm run build              # production build
```

## Development Philosophy

The ERP is built incrementally rather than all at once. Each feature follows the same flow:

```text
Requirement → Design → Database → Migration → Repository
→ Service → Handler → Route → Authorization → Test → Documentation
```

Priorities: correctness, security, maintainability, readable code, separation of concerns, testability, explicit behavior, secure defaults, and simple architecture.
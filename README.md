# ERP System

Monorepo ERP project scaffold with frontend, backend, and infrastructure support.

## Getting started

1. Copy `.env.example` to `.env` and update secrets.
2. Run `make up`.
3. Run this command at root project `docker compose up -d`

## Structure

- `frontend/`: React + TypeScript + Vite
- `backend/`: Go REST API + WebSocket
- `docker-compose.yml`: local dev services

## Features

- JWT-based authentication with access tokens and refresh tokens
- Secure refresh token rotation with opaque token storage and SHA-256 token hashing
- Refresh token reuse detection using refresh token families
- Atomic refresh token family revocation on reuse detection
- User authentication with role and permission support
- Go backend services designed for REST API, RBAC, and extensibility
- Monorepo structure with separate frontend, backend, and infrastructure layers

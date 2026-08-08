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

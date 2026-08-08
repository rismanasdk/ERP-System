.PHONY: up down build fmt test

up:
	docker compose up --build

down:
	docker compose down

build:
	docker compose build

fmt:
	cd backend && gofmt -w .
	cd frontend && npm run lint || true

test:
	@echo "No tests configured yet"

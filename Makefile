include .env
export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_HOST

DATABASE_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):5432/$(POSTGRES_DB)?sslmode=disable

MIGRATE=migrate -path ./migrations -database $(DATABASE_URL)


.PHONY: migrate-up migrate-down migrate-version migrate-force

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-version:
	$(MIGRATE) version

migrate-force:
	$(MIGRATE) force $(ver)
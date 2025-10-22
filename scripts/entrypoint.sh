#!/bin/sh
set -e

DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}?sslmode=disable

echo "Starting migrations..."

/usr/local/bin/migrate -path /root/migrations -database "$DATABASE_URL" up

echo "Starting app..."
exec ./app
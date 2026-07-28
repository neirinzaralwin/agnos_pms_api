POSTGRES_VOLUME=pms_postgres_data
CERT_DIR=nginx/certs
MIGRATE_IMAGE=migrate/migrate:v4.18.1

# Load .env for migrate targets when present.
ifneq (,$(wildcard .env))
include .env
export
endif

DATABASE_URL_LOCAL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@127.0.0.1:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

setup: certs
	docker volume create $(POSTGRES_VOLUME)

certs:
	@mkdir -p $(CERT_DIR)
	@[ -f $(CERT_DIR)/server.crt ] || openssl req -x509 -nodes -newkey rsa:2048 \
		-keyout $(CERT_DIR)/server.key -out $(CERT_DIR)/server.crt -days 365 \
		-subj "/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,DNS:nginx,IP:127.0.0.1"

up: setup
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

migrate:
	docker run --rm --network host \
		-v "$(CURDIR)/migrations:/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path /migrations \
		-database "$(DATABASE_URL_LOCAL)" \
		up

migrate-down:
	docker run --rm --network host \
		-v "$(CURDIR)/migrations:/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path /migrations \
		-database "$(DATABASE_URL_LOCAL)" \
		down 1

migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=description" && exit 1)
	@next=$$(printf "%04d" $$(( $$(ls migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' ') + 1 ))); \
		touch "migrations/$${next}_$(NAME).up.sql" "migrations/$${next}_$(NAME).down.sql"; \
		echo "Created migrations/$${next}_$(NAME).{up,down}.sql"

test:
	go test ./... -race -cover

.PHONY: setup certs up down logs ps migrate migrate-down migrate-create test

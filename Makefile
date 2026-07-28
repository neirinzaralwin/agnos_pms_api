POSTGRES_VOLUME=pms_postgres_data
CERT_DIR=nginx/certs

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

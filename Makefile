infra/up:
	docker compose up -d mysql mysqld-exporter prometheus

infra/down:
	docker compose down mysql mysqld-exporter prometheus

infra/up:
	docker compose up -d mysql mysqld-exporter prometheus

infra/down:
	docker compose down mysql mysqld-exporter prometheus

kafka/up:
	docker compose up -d kafka

kafka/down:
	docker compose down kafka

run:
	go run ./cmd/api

kafka/setup:
	docker compose exec kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic messages \
		--partitions 1 \
		--replication-factor 1

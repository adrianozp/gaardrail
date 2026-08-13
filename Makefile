kafka/up:
	docker compose up -d kafka

kafka/down:
	docker compose down kafka

kafka/setup:
	docker compose exec kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic messages \
		--partitions 1 \
		--replication-factor 1

run:
	go run ./cmd/api

docker/build:
	docker build -t gaardrail .

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

flood/up:
	docker compose -f flood-test/docker-compose-flood.yml up -d

flood/down:
	docker compose -f flood-test/docker-compose-flood.yml down

flood/setup:
	./flood-test/scripts/setup-db.sh 1

flood/messages:
	./flood-test/scripts/flood.sh 10000

run:
	go run ./cmd/api

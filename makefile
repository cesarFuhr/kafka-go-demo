setup:
	docker compose up -d kafka

up:
	docker compose up --force-recreate --build -d producer consumer_1 consumer_2 retrier retrier_producer

down:
	docker compose down

logs:
	docker compose logs -f producer consumer_1 consumer_2 retrier retrier_producer

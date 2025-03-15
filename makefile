setup:
	docker compose up -d kafka

up:
	docker compose up --build -d producer consumer retrier

down:
	docker compose down

logs:
	docker compose logs -f producer consumer retrier

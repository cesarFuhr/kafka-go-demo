FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 go build -mod=vendor -o main ./cmd/app/main.go

# Base container.
FROM scratch AS base
COPY --from=builder /src/main /bin/main

# Publisher container.
FROM base AS producer
ENTRYPOINT [ "/bin/main", "producer" ]

# Consumer container.
FROM base AS consumer
ENTRYPOINT [ "/bin/main", "consumer" ]

# Retrier container.
FROM base AS retrier
ENTRYPOINT [ "/bin/main", "retrier" ]

# Retrier producer container.
FROM base AS retrier_producer
ENTRYPOINT [ "/bin/main", "retrier_producer" ]

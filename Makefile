.PHONY: run test vet tidy build docker-up docker-down
run:
	go run ./cmd/server
test:
	go test ./...
vet:
	go vet ./...
tidy:
	go mod tidy
build:
	go build -o bin/data-profiler ./cmd/server
docker-up:
	docker compose up --build
docker-down:
	docker compose down

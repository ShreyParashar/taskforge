.PHONY: build test vet fmt lint clean run-api migrate-up migrate-down docker-up docker-down

# Build all binaries
build:
	go build -o bin/taskforge-api ./cmd/api
	go build -o bin/taskforge-worker ./cmd/worker
	go build -o bin/taskforge-scheduler ./cmd/scheduler
	go build -o bin/taskforge-reaper ./cmd/reaper

# Build only the API server
build-api:
	go build -o bin/taskforge-api ./cmd/api

# Run the API server locally
run-api:
	go run ./cmd/api

# Run all tests
test:
	go test -v -race -count=1 ./...

# Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Vet the code
vet:
	go vet ./...

# Format the code
fmt:
	gofmt -s -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker Compose
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Database migrations (requires goose)
migrate-up:
	goose -dir migrations postgres "$(TASKFORGE_DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(TASKFORGE_DATABASE_URL)" down
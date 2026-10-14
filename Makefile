.PHONY: build test frontend migrate init-users run
frontend:
	cd frontend && npm ci && npm run build
build: frontend
	go build -o bin/kzcrm ./cmd/crm
test:
	go test ./...
migrate:
	go run ./cmd/migrate
init-users:
	go run ./cmd/init-users
run:
	go run ./cmd/crm

build:
	@go build -o ./bin/plunger-server ./cmd/plunger-server

test:
	@go test ./...
	
run: build
	@./bin/plunge-server

migrate-up:
	@./scripts/goose.sh

migrate-down:
	@./scripts/goose-down.sh

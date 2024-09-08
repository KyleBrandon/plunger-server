build:
	@go build -o bin/plunge-server

test:
	@go test -v ./...
	
run: build
	@./bin/plunge-server

migrate-up:
	@goose postgres postgres://postgres:postgres@10.0.4.40:5444/plunger up

migrate-down:
	@goose postgres postgres://postgres:postgres@10.0.4.40:5444/plunger down

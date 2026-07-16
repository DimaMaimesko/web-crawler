tidy:
	go mod tidy

test:
	go test -race ./...

run:
	go run ./cmd/crawler/

run-race:
	go run -race ./cmd/crawler/

build:
	go build -o bin/crawler ./cmd/crawler/
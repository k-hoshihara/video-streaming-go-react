.PHONY: generate test lint run docker-build

# docs/api/v1/openapi.yaml からコードを再生成する
generate:
	go generate ./...

test:
	go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t video-streaming-backend .

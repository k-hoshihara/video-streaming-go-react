.PHONY: generate test coverage lint run docker-build

# docs/api/v1/openapi.yaml からコードを再生成する
generate:
	go generate ./...

# 自動生成コード（internal/oapi）はカバレッジの計測対象から外す
COVER_PKGS := $(shell go list ./internal/... | grep -v '/internal/oapi$$' | paste -sd, -)

test:
	go test -race -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# 命令網羅（C0）と分岐網羅（C1）を出力し、カバレッジを色分けした HTML を開く
coverage: test
	scripts/branch-coverage.sh branch-coverage.json
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t video-streaming-backend .

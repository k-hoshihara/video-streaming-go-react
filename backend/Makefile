.PHONY: generate test coverage lint run docker-build

# docs/api/v1/openapi.yaml からコードを再生成する
generate:
	go generate ./...

# 自動生成コード（internal/oapi）とテスト用のヘルパー（internal/testdb）はカバレッジの計測対象から外す
COVER_PKGS := $(shell go list ./internal/... | grep -v -E '/internal/(oapi|testdb)$$' | paste -sd, -)

test:
	go test -race -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# 命令網羅（C0）と分岐網羅（C1）を出力し、カバレッジを色分けした HTML を開く
coverage: test
	scripts/branch-coverage.sh branch-coverage.json
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

# ローカルの DB（db/compose.yml）に接続して起動する
DATABASE_URL ?= postgres://app:app@localhost:5432/app?sslmode=disable

run:
	DATABASE_URL='$(DATABASE_URL)' go run ./cmd/server

docker-build:
	docker build -t video-streaming-backend .

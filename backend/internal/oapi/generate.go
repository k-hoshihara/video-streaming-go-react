// Package oapi は OpenAPI の仕様書から生成したコードを置くパッケージ。
// 仕様書（リポジトリ直下の docs/api/v1/openapi.yaml）を更新したら backend/ で `go generate ./...` を実行して再生成する。
package oapi

//go:generate go tool oapi-codegen -config config.yaml ../../../docs/api/v1/openapi.yaml

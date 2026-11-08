# backend

Go（Echo）で作った API サーバーです。

## 自動生成のコードと手書きのコード

自動生成されるのは `internal/oapi/oapi.gen.go` だけです。それ以外はすべて手で書いています。

```
backend/
├── cmd/server/            手書き    起動処理（部品の組み立て）
└── internal/
    ├── oapi/
    │   ├── oapi.gen.go    自動生成  仕様書から生成した型・ルーティング（手で編集しない）
    │   ├── generate.go    手書き    生成コマンドの指示
    │   └── config.yaml    手書き    生成の設定
    ├── server/            手書き    Echo の設定、仕様書による検証、エラー処理
    ├── handler/           手書き    HTTP と usecase の間の変換
    ├── usecase/           手書き    業務ロジック
    ├── domain/            手書き    エンティティとエラー
    └── repository/        手書き    データの保存（postgres / memory）
```

手書きのコードと自動生成のコードは、次のようにつながっています。

- 自動生成のコードには、仕様書から作られたリクエストやレスポンスの型と、`StrictServerInterface` というインターフェースがあります。
- 手書きの `handler` は、このインターフェースを実装します。
- エンドポイントを増やすと、`StrictServerInterface` にメソッドが追加されます。`handler` に実装を足さないとコンパイルエラーになるため、実装し忘れに気づけます。

## コードの自動生成

API の型やルーティングは、仕様書 [`docs/api/v1/openapi.yaml`](../docs/api/v1/openapi.yaml) から [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) で自動生成しています。

| ファイル | 種類 | 役割 |
|---|---|---|
| `docs/api/v1/openapi.yaml` | 手書き | API の仕様書。API を変えるときはここを編集する |
| `internal/oapi/generate.go` | 手書き | `go generate` で実行するコマンド |
| `internal/oapi/config.yaml` | 手書き | oapi-codegen の設定（生成するコードの種類など） |
| `internal/oapi/oapi.gen.go` | 自動生成 | 生成されたコード。手で編集しない |

### 生成のしかた

仕様書を変更したら、`backend/` で次のどちらかを実行し、生成されたコードも一緒にコミットしてください。

```sh
make generate
# または
go generate ./...
```

oapi-codegen は `go.mod` の `tool` に登録しているため、別途インストールする必要はありません。

### CI での確認

CI（`.github/workflows/backend-ci.yml`）は、コードを生成し直したうえで `git diff --exit-code` を実行します。次のような場合は差分が出るため、CI が失敗します。

- 仕様書を変更したのに、コードを生成し直していない
- 生成されたコードを手で編集した

失敗したときは、上の手順でコードを生成し直してコミットしてください。

### 生成コードをコミットする理由

- clone した直後でも、生成の手順なしにビルドできる
- PR で、仕様書の変更が生成コードにどう反映されたかを確認できる

なお、生成されたコードはカバレッジの計測対象から外しています。

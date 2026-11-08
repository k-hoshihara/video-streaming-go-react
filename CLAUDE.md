# CLAUDE.md

## Git / GitHub

- コミットメッセージと PR の説明欄に、Claude や AI が作成したことを示す記載を入れないこと
  - 例: `Co-Authored-By: Claude ...`、`Claude-Session: ...`、`🤖 Generated with [Claude Code](...)`、claude.ai のセッション URL

## テスト

- 条件分岐を追加・変更したら、すべての分岐を通るテストを書くこと
- 長さ・数値・件数などに上限や下限がある入力は、必ず境界値のテストを書くこと
  - 境界の内側と外側の両方を確認する（例: 1〜50 文字なら 0・1・50・51 文字）
  - 文字数とバイト数で制限が異なるもの（例: bcrypt の 72 バイト）は、マルチバイト文字でも確認する
- repository のテストと、HTTP から DB までを通すテスト（`internal/server`）は、偽物やメモリ上の repository ではなく本物の PostgreSQL を使うこと
  - `internal/testdb` の `testdb.New(t)` で、テストごとに専用のスキーマを用意する
  - usecase と handler の単体テストは、その層だけを確かめるため偽物を使ってよい

## DB マイグレーション

- `backend/db/migrations/` の SQL ファイル名には、DDL か DML かを含めること
  - 形式: `<連番3桁>_<ddl|dml>_<内容>.sql`（例: `001_ddl_create_users.sql`、`002_dml_insert_plans.sql`）
  - 番号順に実行されるため、連番を先頭に置く

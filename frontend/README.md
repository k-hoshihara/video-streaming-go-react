# frontend

React 19 + TypeScript 7 + Vite 8 のフロントエンド。

## 必要なもの

- Node.js 26（LTS）以上（`.node-version` を参照）

## 開発

```sh
npm install
npm run dev      # http://localhost:5173
```

開発サーバーは `/api/*` を API サーバー（既定は `http://localhost:8080`）へ中継する。
中継先は環境変数 `API_PROXY_TARGET` で変更できる。

```sh
# 別ターミナルで API サーバーを起動しておく
cd ../backend && go run ./cmd/server
```

本番などで API を別オリジンから呼ぶ場合は、ビルド時に `VITE_API_BASE_URL` を設定する。

## コマンド

| コマンド | 内容 |
| --- | --- |
| `npm run dev` | 開発サーバーを起動 |
| `npm run build` | 型チェックとビルド |
| `npm test` | テスト（Vitest） |
| `npm run lint` | Lint（oxlint） |

## API の型

`src/api/types.ts` は `backend/api/openapi.yaml` に合わせて手で書いている。
（`openapi-typescript` が TypeScript 7 に未対応のため、型の自動生成は使っていない）

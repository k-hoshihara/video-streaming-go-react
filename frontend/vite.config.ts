/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // 開発時は /api へのリクエストを API サーバーに中継する（CORS を不要にする）。
    proxy: {
      '/api': {
        target: process.env.API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      // エントリーポイント（画面の組み立てのみ）とテスト用のファイルは計測対象から外す
      exclude: ['src/main.tsx', 'src/App.tsx', 'src/**/*.test.{ts,tsx}', 'src/test/**', 'src/**/*.d.ts'],
      reporter: ['text', 'json-summary'],
      // CLAUDE.md の「すべての分岐を通るテストを書く」に合わせ、分岐網羅が 100% 未満なら失敗にする
      thresholds: { branches: 100 },
    },
  },
})

import type { ApiErrorBody } from './types'

// API のベース URL。未設定時は Vite の開発サーバーのプロキシ（/api）を使う。
const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

// ApiError は API が 2xx 以外を返したときのエラー。
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function postJSON<T>(path: string, body: unknown, signal?: AbortSignal): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  })

  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as Partial<ApiErrorBody> | null
    throw new ApiError(res.status, data?.message ?? res.statusText)
  }
  return (await res.json()) as T
}

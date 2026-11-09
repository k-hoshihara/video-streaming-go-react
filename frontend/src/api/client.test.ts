import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, postJSON } from './client'

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockReset()
})

describe('postJSON', () => {
  it('2xx なら JSON を返す', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 201 }))

    await expect(postJSON('/v1/x', { a: 1 })).resolves.toEqual({ ok: true })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/x')
    expect(init?.method).toBe('POST')
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
    expect(init?.body).toBe('{"a":1}')
  })

  it('2xx 以外なら message を持つ ApiError を投げる', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ message: 'email already exists' }), { status: 409 }),
    )

    const err = await postJSON('/v1/x', {}).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 409, message: 'email already exists' })
  })

  it('エラーの本文が JSON でなければ statusText を message にする', async () => {
    fetchMock.mockResolvedValue(
      new Response('<html>Bad Gateway</html>', { status: 502, statusText: 'Bad Gateway' }),
    )

    await expect(postJSON('/v1/x', {})).rejects.toMatchObject({
      status: 502,
      message: 'Bad Gateway',
    })
  })

  it('エラーの本文に message がなければ statusText を message にする', async () => {
    fetchMock.mockResolvedValue(
      new Response('{}', { status: 500, statusText: 'Internal Server Error' }),
    )

    await expect(postJSON('/v1/x', {})).rejects.toMatchObject({
      status: 500,
      message: 'Internal Server Error',
    })
  })

  it('VITE_API_BASE_URL があればそれをベース URL にする', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com')
    vi.resetModules()
    const { postJSON: postWithEnv } = await import('./client')
    fetchMock.mockResolvedValue(new Response('{}', { status: 200 }))

    await postWithEnv('/v1/x', {})

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.example.com/v1/x')
    vi.unstubAllEnvs()
  })
})

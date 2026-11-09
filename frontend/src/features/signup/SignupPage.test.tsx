import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SignupPage } from './SignupPage'

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockReset()
})

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

async function fillAndSubmit(values: { name?: string; email?: string; password?: string }) {
  const user = userEvent.setup()
  render(<SignupPage />)
  if (values.name) await user.type(screen.getByLabelText('ユーザー名'), values.name)
  if (values.email) await user.type(screen.getByLabelText('メールアドレス'), values.email)
  if (values.password) await user.type(screen.getByLabelText('パスワード'), values.password)
  await user.click(screen.getByRole('button', { name: '登録する' }))
}

describe('SignupPage', () => {
  it('登録に成功したら完了画面を表示する', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(201, {
        id: '0190a0e0-0000-7000-8000-000000000000',
        name: 'taro',
        email: 'taro@example.com',
        created_at: '2026-11-08T00:00:00Z',
      }),
    )

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(await screen.findByRole('heading', { name: '登録が完了しました' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledOnce()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/signup')
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({
      name: 'taro',
      email: 'taro@example.com',
      password: 'password123',
    })
  })

  it('入力が不正なら API を呼ばずにエラーを表示する', async () => {
    await fillAndSubmit({ name: 'taro', email: 'invalid', password: 'short' })

    expect(await screen.findByText('メールアドレスの形式が正しくありません')).toBeInTheDocument()
    expect(screen.getByText('パスワードは8文字以上で入力してください')).toBeInTheDocument()
    expect(screen.getByLabelText('メールアドレス')).toHaveAttribute('aria-invalid', 'true')
    expect(fetchMock).not.toHaveBeenCalled()
    // 入力値は保持する
    expect(screen.getByLabelText('ユーザー名')).toHaveValue('taro')
    expect(screen.getByLabelText('メールアドレス')).toHaveValue('invalid')
  })

  it('メールアドレスが登録済み（409）ならメールアドレス欄にエラーを表示する', async () => {
    fetchMock.mockResolvedValue(jsonResponse(409, { message: 'email already exists' }))

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(
      await screen.findByText('このメールアドレスはすでに登録されています'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('メールアドレス')).toHaveValue('taro@example.com')
  })

  it('リクエストが不正（400）なら全体のエラーを表示する', async () => {
    fetchMock.mockResolvedValue(jsonResponse(400, { message: 'password is too long' }))

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(await screen.findByRole('alert')).toHaveTextContent('入力内容を確認してください')
  })

  it('メールアドレスの前後の空白を除いて送信する', async () => {
    fetchMock.mockResolvedValue(jsonResponse(409, { message: 'email already exists' }))

    await fillAndSubmit({ name: 'taro', email: '  taro@example.com  ', password: 'password123' })

    await screen.findByText('このメールアドレスはすでに登録されています')
    expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body)).email).toBe('taro@example.com')
  })

  it('送信中はボタンを押せない', async () => {
    let resolve!: (res: Response) => void
    fetchMock.mockReturnValue(new Promise((r) => (resolve = r)))

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(await screen.findByRole('button', { name: '登録中…' })).toBeDisabled()
    resolve(jsonResponse(409, { message: 'email already exists' }))
    expect(await screen.findByRole('button', { name: '登録する' })).toBeEnabled()
  })

  it('パスワード欄はエラーがなければヒントを、あればエラーを説明として結び付ける', async () => {
    const user = userEvent.setup()
    render(<SignupPage />)

    const password = screen.getByLabelText('パスワード')
    expect(password).toHaveAccessibleDescription('8文字以上')
    expect(password).not.toHaveAttribute('aria-invalid')
    // ヒントのない欄は説明を持たない
    expect(screen.getByLabelText('ユーザー名')).not.toHaveAttribute('aria-describedby')

    await user.click(screen.getByRole('button', { name: '登録する' }))

    expect(await screen.findByLabelText('パスワード')).toHaveAccessibleDescription(
      'パスワードを入力してください',
    )
    expect(screen.queryByText('8文字以上')).not.toBeInTheDocument()
    expect(screen.getByLabelText('ユーザー名')).toHaveAccessibleDescription(
      'ユーザー名を入力してください',
    )
  })

  it('サーバーエラーなら全体のエラーを表示する', async () => {
    fetchMock.mockResolvedValue(jsonResponse(500, { message: 'Internal Server Error' }))

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '登録に失敗しました。時間をおいて再度お試しください',
    )
  })

  it('通信に失敗したら全体のエラーを表示する', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))

    await fillAndSubmit({ name: 'taro', email: 'taro@example.com', password: 'password123' })

    expect(await screen.findByRole('alert')).toHaveTextContent('登録に失敗しました')
  })
})

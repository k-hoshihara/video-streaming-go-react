import { describe, expect, it, vi } from 'vitest'
import { initialSignupState, signupAction } from './signupAction'

describe('signupAction', () => {
  it('フォームに項目がなければ空文字として検証し、API を呼ばない', async () => {
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal('fetch', fetchMock)

    const state = await signupAction(initialSignupState, new FormData())

    expect(state).toEqual({
      status: 'error',
      values: { name: '', email: '', password: '' },
      fieldErrors: {
        name: 'ユーザー名を入力してください',
        email: 'メールアドレスを入力してください',
        password: 'パスワードを入力してください',
      },
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

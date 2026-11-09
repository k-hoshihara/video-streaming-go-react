import { signup } from '../../api/auth'
import { ApiError } from '../../api/client'
import type { SignupRequest, User } from '../../api/types'
import { type SignupFieldErrors, validateSignup } from './validation'

export type SignupState =
  | { status: 'idle' | 'error'; values: SignupRequest; fieldErrors: SignupFieldErrors; formError?: string }
  | { status: 'success'; user: User }

export const initialSignupState: SignupState = {
  status: 'idle',
  values: { name: '', email: '', password: '' },
  fieldErrors: {},
}

// signupAction はフォームの送信を処理する（useActionState に渡す）。
export async function signupAction(_prev: SignupState, formData: FormData): Promise<SignupState> {
  const values: SignupRequest = {
    name: String(formData.get('name') ?? ''),
    email: String(formData.get('email') ?? '').trim(),
    password: String(formData.get('password') ?? ''),
  }

  const fieldErrors = validateSignup(values)
  if (Object.keys(fieldErrors).length > 0) {
    return { status: 'error', values, fieldErrors }
  }

  try {
    const user = await signup(values)
    return { status: 'success', user }
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      return {
        status: 'error',
        values,
        fieldErrors: { email: 'このメールアドレスはすでに登録されています' },
      }
    }
    if (err instanceof ApiError && err.status === 400) {
      return { status: 'error', values, fieldErrors: {}, formError: '入力内容を確認してください' }
    }
    return {
      status: 'error',
      values,
      fieldErrors: {},
      formError: '登録に失敗しました。時間をおいて再度お試しください',
    }
  }
}

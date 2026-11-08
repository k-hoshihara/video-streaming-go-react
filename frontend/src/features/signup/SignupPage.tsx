import { useActionState, useId } from 'react'
import { ApiError } from '../../api/client'
import { signup } from '../../api/auth'
import type { SignupRequest, User } from '../../api/types'
import { type SignupFieldErrors, validateSignup } from './validation'
import './SignupPage.css'

type SignupState =
  | { status: 'idle' | 'error'; values: SignupRequest; fieldErrors: SignupFieldErrors; formError?: string }
  | { status: 'success'; user: User }

const initialState: SignupState = {
  status: 'idle',
  values: { name: '', email: '', password: '' },
  fieldErrors: {},
}

async function signupAction(_prev: SignupState, formData: FormData): Promise<SignupState> {
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

export function SignupPage() {
  const [state, formAction, isPending] = useActionState(signupAction, initialState)
  const id = useId()

  if (state.status === 'success') {
    return (
      <main className="signup">
        <section className="signup-card" aria-labelledby={`${id}-done`}>
          <h1 id={`${id}-done`}>登録が完了しました</h1>
          <p>
            ようこそ、<strong>{state.user.name}</strong> さん。
            <br />
            {state.user.email} で登録しました。
          </p>
        </section>
      </main>
    )
  }

  const { values, fieldErrors, formError } = state
  const fields = [
    { name: 'name', label: 'ユーザー名', type: 'text', autoComplete: 'username' },
    { name: 'email', label: 'メールアドレス', type: 'email', autoComplete: 'email' },
    {
      name: 'password',
      label: 'パスワード',
      type: 'password',
      autoComplete: 'new-password',
      hint: '8文字以上',
    },
  ] as const

  return (
    <main className="signup">
      <section className="signup-card" aria-labelledby={`${id}-title`}>
        <h1 id={`${id}-title`}>アカウント登録</h1>

        {formError && (
          <p className="signup-form-error" role="alert">
            {formError}
          </p>
        )}

        {/* 入力値の検証は validateSignup で行い、エラー表示を統一するため noValidate にする */}
        <form action={formAction} noValidate>
          {fields.map((f) => {
            const inputId = `${id}-${f.name}`
            const error = fieldErrors[f.name]
            const hint = 'hint' in f ? f.hint : undefined
            const describedBy = error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined
            return (
              <div className="signup-field" key={f.name}>
                <label htmlFor={inputId}>{f.label}</label>
                <input
                  id={inputId}
                  name={f.name}
                  type={f.type}
                  autoComplete={f.autoComplete}
                  defaultValue={values[f.name]}
                  aria-invalid={error ? true : undefined}
                  aria-describedby={describedBy}
                  required
                />
                {error ? (
                  <p id={`${inputId}-error`} className="signup-field-error">
                    {error}
                  </p>
                ) : (
                  hint && (
                    <p id={`${inputId}-hint`} className="signup-field-hint">
                      {hint}
                    </p>
                  )
                )}
              </div>
            )
          })}

          <button type="submit" disabled={isPending}>
            {isPending ? '登録中…' : '登録する'}
          </button>
        </form>
      </section>
    </main>
  )
}

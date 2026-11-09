import type { SignupRequest } from '../../api/types'

// 制約は docs/api/v1/openapi.yaml の SignupRequest と backend/internal/domain/user.go に合わせる。
export const NAME_MAX_LENGTH = 50
export const EMAIL_MAX_LENGTH = 254
export const PASSWORD_MIN_LENGTH = 8
// bcrypt は 72 バイトを超えるパスワードを扱えないため、バイト数で上限を確認する。
export const PASSWORD_MAX_BYTES = 72

export type SignupFieldErrors = Partial<Record<keyof SignupRequest, string>>

// JSON Schema の minLength / maxLength はコードポイント単位で数える。
const charLength = (s: string) => [...s].length
const byteLength = (s: string) => new TextEncoder().encode(s).length

// 制御文字（\p{Cc}）と行・段落の区切り（\p{Zl}、\p{Zp}）は使えない。
const NAME_PATTERN = /^[^\p{Cc}\p{Zl}\p{Zp}]*$/u

// 仕様書の email の pattern と同じ。
// ブラウザの <input type="email"> の形式に、@ の前は 64 文字まで・ドメインに . が必要、を加えたもの。
const EMAIL_PATTERN =
  /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]{1,64}@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$/

// @ の前の . は先頭・末尾・連続を不可（バックエンドは net/mail.ParseAddress で検証する）。
const hasInvalidLocalDots = (email: string) => {
  const local = email.slice(0, email.lastIndexOf('@'))
  return local.startsWith('.') || local.endsWith('.') || local.includes('..')
}

export function validateSignup(values: SignupRequest): SignupFieldErrors {
  const errors: SignupFieldErrors = {}

  if (values.name.trim() === '') {
    errors.name = 'ユーザー名を入力してください'
  } else if (charLength(values.name) > NAME_MAX_LENGTH) {
    errors.name = `ユーザー名は${NAME_MAX_LENGTH}文字以内で入力してください`
  } else if (!NAME_PATTERN.test(values.name)) {
    errors.name = 'ユーザー名に改行や制御文字は使えません'
  }

  if (values.email === '') {
    errors.email = 'メールアドレスを入力してください'
  } else if (charLength(values.email) > EMAIL_MAX_LENGTH) {
    errors.email = `メールアドレスは${EMAIL_MAX_LENGTH}文字以内で入力してください`
  } else if (!EMAIL_PATTERN.test(values.email) || hasInvalidLocalDots(values.email)) {
    errors.email = 'メールアドレスの形式が正しくありません'
  }

  if (values.password === '') {
    errors.password = 'パスワードを入力してください'
  } else if (charLength(values.password) < PASSWORD_MIN_LENGTH) {
    errors.password = `パスワードは${PASSWORD_MIN_LENGTH}文字以上で入力してください`
  } else if (byteLength(values.password) > PASSWORD_MAX_BYTES) {
    errors.password = 'パスワードが長すぎます（半角72文字以内）'
  }

  return errors
}

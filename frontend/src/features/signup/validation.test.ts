import { describe, expect, it } from 'vitest'
import { validateSignup } from './validation'

const valid = { name: 'taro', email: 'taro@example.com', password: 'password123' }

describe('validateSignup', () => {
  it('正しい入力ならエラーなし', () => {
    expect(validateSignup(valid)).toEqual({})
  })

  it.each([
    ['name', '', 'ユーザー名を入力してください'],
    ['name', '   ', 'ユーザー名を入力してください'],
    ['name', 'あ'.repeat(51), 'ユーザー名は50文字以内で入力してください'],
    ['email', '', 'メールアドレスを入力してください'],
    ['email', 'taro', 'メールアドレスの形式が正しくありません'],
    ['email', `${'a'.repeat(250)}@example.com`, 'メールアドレスは254文字以内で入力してください'],
    ['password', '', 'パスワードを入力してください'],
    ['password', 'a'.repeat(7), 'パスワードは8文字以上で入力してください'],
    ['password', 'a'.repeat(73), 'パスワードが長すぎます（半角72文字以内）'],
    // 全角 25 文字 = 75 バイト
    ['password', 'あ'.repeat(25), 'パスワードが長すぎます（半角72文字以内）'],
  ] as const)('%s=%j のとき「%s」', (field, value, message) => {
    expect(validateSignup({ ...valid, [field]: value })).toEqual({ [field]: message })
  })

  it('境界値はエラーなし', () => {
    expect(
      validateSignup({ name: 'あ'.repeat(50), email: valid.email, password: 'a'.repeat(72) }),
    ).toEqual({})
    expect(validateSignup({ ...valid, password: 'a'.repeat(8) })).toEqual({})
  })
})

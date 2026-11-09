import { describe, expect, it } from 'vitest'
import { validateSignup } from './validation'

const valid = { name: 'taro', email: 'taro@example.com', password: 'password123' }

const errorOf = (field: keyof typeof valid, value: string) =>
  validateSignup({ ...valid, [field]: value })[field]

describe('validateSignup', () => {
  it('正しい入力ならエラーなし', () => {
    expect(validateSignup(valid)).toEqual({})
  })

  it('複数の項目が不正なら、すべての項目のエラーを返す', () => {
    expect(validateSignup({ name: '', email: '', password: '' })).toEqual({
      name: 'ユーザー名を入力してください',
      email: 'メールアドレスを入力してください',
      password: 'パスワードを入力してください',
    })
  })

  describe('name（1〜50 文字）', () => {
    it.each([
      ['0 文字', ''],
      ['空白のみ', '   '],
    ])('%s は NG', (_, value) => {
      expect(errorOf('name', value)).toBe('ユーザー名を入力してください')
    })

    it.each([
      ['1 文字', 'a'],
      ['50 文字', 'a'.repeat(50)],
      ['マルチバイト 50 文字', 'あ'.repeat(50)],
      ['絵文字 50 文字（サロゲートペアも 1 文字と数える）', '😀'.repeat(50)],
      ['途中の空白', 'taro yamada'],
    ])('%s は OK', (_, value) => {
      expect(errorOf('name', value)).toBeUndefined()
    })

    it.each([
      ['51 文字', 'a'.repeat(51)],
      ['マルチバイト 51 文字', 'あ'.repeat(51)],
    ])('%s は NG', (_, value) => {
      expect(errorOf('name', value)).toBe('ユーザー名は50文字以内で入力してください')
    })

    it.each([
      ['改行（LF）', 'ta\nro'],
      ['改行（CR）', 'ta\rro'],
      ['タブ', 'ta\tro'],
      ['NUL', 'ta\u0000ro'],
      ['DEL', 'ta\u007fro'],
      ['NEL（C1 制御文字）', 'ta\u0085ro'],
      ['行の区切り（U+2028）', 'ta ro'],
      ['段落の区切り（U+2029）', 'ta ro'],
    ])('%s を含むと NG', (_, value) => {
      expect(errorOf('name', value)).toBe('ユーザー名に改行や制御文字は使えません')
    })
  })

  describe('email', () => {
    it('0 文字は NG', () => {
      expect(errorOf('email', '')).toBe('メールアドレスを入力してください')
    })

    // 全体 254 文字: 64 + 1 + (63 + 1) * 2 + 61 = 254
    const label63 = 'b'.repeat(63)
    const email254 = `${'a'.repeat(64)}@${label63}.${label63}.${'c'.repeat(61)}`

    it('全体 254 文字は OK', () => {
      expect(email254).toHaveLength(254)
      expect(errorOf('email', email254)).toBeUndefined()
    })

    it('全体 255 文字は NG', () => {
      const email255 = `${email254}c`
      expect(email255).toHaveLength(255)
      expect(errorOf('email', email255)).toBe('メールアドレスは254文字以内で入力してください')
    })

    it.each([
      'taro@example.com',
      'Taro@Example.com',
      'taro+test@example.com',
      'taro.yamada@mail.example.co.jp',
      'taro@my-example.com',
      '1234@example.com',
      `${'a'.repeat(64)}@example.com`, // @ の前 64 文字
      `taro@${'a'.repeat(63)}.com`, // ラベル 63 文字
    ])('%s は OK', (value) => {
      expect(errorOf('email', value)).toBeUndefined()
    })

    it.each([
      'taro',
      'taro@localhost',
      '山田@example.jp',
      'taro@例え.jp',
      '😀@example.com',
      'ta ro@example.com',
      '"taro"@example.com',
      'taro@[192.168.0.1]',
      'taro@@example.com',
      '@example.com',
      'taro@',
      'taro@-example.com',
      'taro@example-.com',
      'taro@example..com',
      '.taro@example.com',
      'taro.@example.com',
      'ta..ro@example.com',
      `${'a'.repeat(65)}@example.com`, // @ の前 65 文字
      `taro@${'a'.repeat(64)}.com`, // ラベル 64 文字
    ])('%s は NG', (value) => {
      expect(errorOf('email', value)).toBe('メールアドレスの形式が正しくありません')
    })
  })

  describe('password（8 文字以上・72 バイト以内）', () => {
    it('0 文字は NG', () => {
      expect(errorOf('password', '')).toBe('パスワードを入力してください')
    })

    it.each([
      ['7 文字', 'a'.repeat(7)],
      ['マルチバイト 7 文字', 'あ'.repeat(7)],
    ])('%s は NG', (_, value) => {
      expect(errorOf('password', value)).toBe('パスワードは8文字以上で入力してください')
    })

    it.each([
      ['8 文字', 'a'.repeat(8)],
      ['72 バイト（半角 72 文字）', 'a'.repeat(72)],
      ['72 バイト（全角 24 文字）', 'あ'.repeat(24)],
      ['72 バイト（全角 23 文字 + 半角 3 文字）', `${'あ'.repeat(23)}aaa`],
    ])('%s は OK', (_, value) => {
      expect(errorOf('password', value)).toBeUndefined()
    })

    it.each([
      ['73 バイト（半角 73 文字）', 'a'.repeat(73)],
      ['73 バイト（全角 24 文字 + 半角 1 文字）', `${'あ'.repeat(24)}a`],
      ['75 バイト（全角 25 文字）', 'あ'.repeat(25)],
    ])('%s は NG', (_, value) => {
      expect(errorOf('password', value)).toBe('パスワードが長すぎます（半角72文字以内）')
    })
  })
})

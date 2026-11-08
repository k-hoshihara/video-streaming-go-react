import { postJSON } from './client'
import type { SignupRequest, User } from './types'

// signup は POST /signup でユーザーを登録する。
export function signup(req: SignupRequest, signal?: AbortSignal): Promise<User> {
  return postJSON<User>('/signup', req, signal)
}

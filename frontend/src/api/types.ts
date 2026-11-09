// docs/api/v1/openapi.yaml の components.schemas に対応する型。
// 仕様書を変更したらこちらも合わせて更新すること。

export type SignupRequest = {
  name: string
  email: string
  password: string
}

export type User = {
  id: string
  name: string
  email: string
  created_at: string
}

export type ApiErrorBody = {
  message: string
}

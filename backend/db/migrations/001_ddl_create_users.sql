CREATE TABLE users (
    id            uuid         PRIMARY KEY,
    -- 上限は仕様書（docs/api/v1/openapi.yaml）の maxLength に合わせる。varchar(n) の n は文字数
    name          varchar(50)  NOT NULL,
    -- アプリケーション側で小文字に揃えてから保存する
    email         varchar(254) NOT NULL UNIQUE,
    password_hash text         NOT NULL,
    created_at    timestamptz  NOT NULL
);

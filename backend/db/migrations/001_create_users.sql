CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    name          text        NOT NULL,
    -- アプリケーション側で小文字に揃えてから保存する
    email         text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL
);

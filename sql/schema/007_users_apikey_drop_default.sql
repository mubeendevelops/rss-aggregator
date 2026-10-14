-- +goose Up

ALTER TABLE users ALTER COLUMN api_key DROP DEFAULT;

-- +goose Down

ALTER TABLE users ALTER COLUMN api_key SET DEFAULT (
    encode(sha256(random()::text::bytea),'hex')
);

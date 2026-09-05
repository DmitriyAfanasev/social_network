-- +goose Up

ALTER TABLE profiles.profiles
    ADD COLUMN IF NOT EXISTS avatar_url text NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE profiles.profiles
    DROP COLUMN IF EXISTS avatar_url;

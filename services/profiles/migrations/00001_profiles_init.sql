-- +goose Up

CREATE SCHEMA IF NOT EXISTS profiles;

CREATE TABLE profiles.profiles (
    user_id uuid PRIMARY KEY,
    handle varchar(32),
    display_name varchar(80) NOT NULL DEFAULT '',
    bio text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT profiles_handle_format_check CHECK (
        handle IS NULL OR handle ~ '^[a-z0-9_]{3,32}$'
    )
);

CREATE UNIQUE INDEX profiles_handle_unique_idx
    ON profiles.profiles (lower(handle))
    WHERE handle IS NOT NULL;
CREATE INDEX profiles_handle_search_idx
    ON profiles.profiles (lower(handle) varchar_pattern_ops)
    WHERE handle IS NOT NULL;

-- +goose Down

DROP TABLE profiles.profiles;

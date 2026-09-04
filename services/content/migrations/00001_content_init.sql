-- +goose Up

CREATE SCHEMA IF NOT EXISTS content;

CREATE TABLE content.posts (
    id uuid PRIMARY KEY,
    author_id uuid NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT posts_body_length_check CHECK (char_length(body) BETWEEN 1 AND 5000)
);
CREATE INDEX posts_author_created_idx ON content.posts (author_id, created_at DESC);
CREATE INDEX posts_created_idx ON content.posts (created_at DESC, id DESC);

-- +goose Down

DROP TABLE content.posts;

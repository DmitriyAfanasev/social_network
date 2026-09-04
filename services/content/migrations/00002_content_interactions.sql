-- +goose Up

CREATE TABLE content.comments (
    id uuid PRIMARY KEY,
    post_id uuid NOT NULL REFERENCES content.posts(id) ON DELETE CASCADE,
    author_id uuid NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT comments_body_length_check CHECK (char_length(body) BETWEEN 1 AND 2000)
);

CREATE INDEX comments_post_created_idx ON content.comments (post_id, created_at ASC, id ASC);

CREATE TABLE content.post_likes (
    post_id uuid NOT NULL REFERENCES content.posts(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX post_likes_post_idx ON content.post_likes (post_id, created_at DESC);

-- +goose Down

DROP TABLE content.post_likes;
DROP TABLE content.comments;

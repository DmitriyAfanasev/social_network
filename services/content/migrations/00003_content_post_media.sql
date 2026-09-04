-- +goose Up

ALTER TABLE content.posts DROP CONSTRAINT posts_body_length_check;
ALTER TABLE content.posts ADD CONSTRAINT posts_body_length_check CHECK (char_length(body) BETWEEN 0 AND 5000);

CREATE TABLE content.post_media (
    post_id uuid NOT NULL REFERENCES content.posts(id) ON DELETE CASCADE,
    media_id uuid NOT NULL,
    position smallint NOT NULL CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, media_id),
    UNIQUE (post_id, position)
);

CREATE INDEX post_media_media_idx ON content.post_media (media_id);

-- +goose Down

DROP TABLE content.post_media;
ALTER TABLE content.posts DROP CONSTRAINT posts_body_length_check;
ALTER TABLE content.posts ADD CONSTRAINT posts_body_length_check CHECK (char_length(body) BETWEEN 1 AND 5000);

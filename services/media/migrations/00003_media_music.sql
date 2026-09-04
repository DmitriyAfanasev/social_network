-- +goose Up

CREATE TABLE media.music_tracks (
    id uuid PRIMARY KEY,
    media_id uuid NOT NULL UNIQUE REFERENCES media.media(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    title varchar(200) NOT NULL,
    artist varchar(120) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX music_tracks_user_created_idx ON media.music_tracks (user_id, created_at DESC, id DESC);

-- +goose Down

DROP TABLE media.music_tracks;

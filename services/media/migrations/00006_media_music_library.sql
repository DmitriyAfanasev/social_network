-- +goose Up

CREATE TABLE media.music_library (
    user_id uuid NOT NULL,
    track_id uuid NOT NULL REFERENCES media.music_tracks(id) ON DELETE CASCADE,
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, track_id)
);

CREATE INDEX music_library_user_added_idx
    ON media.music_library (user_id, added_at DESC, track_id DESC);

-- +goose Down

DROP TABLE media.music_library;

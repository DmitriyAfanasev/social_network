-- +goose Up

CREATE TABLE media.video_assets (
    id uuid PRIMARY KEY,
    media_id uuid NOT NULL UNIQUE REFERENCES media.media(id) ON DELETE CASCADE,
    title varchar(255) NOT NULL DEFAULT 'Видео',
    status varchar(32) NOT NULL DEFAULT 'uploaded',
    duration double precision,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT video_assets_status_check CHECK (status IN ('uploaded', 'processing', 'ready', 'failed'))
);

CREATE TABLE media.video_renditions (
    id uuid PRIMARY KEY,
    video_id uuid NOT NULL REFERENCES media.video_assets(id) ON DELETE CASCADE,
    height integer NOT NULL CHECK (height > 0),
    object_key varchar(512) NOT NULL UNIQUE,
    content_type varchar(255) NOT NULL,
    size bigint NOT NULL CHECK (size > 0),
    duration double precision NOT NULL CHECK (duration >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_id, height)
);

CREATE INDEX video_assets_status_created_idx ON media.video_assets (status, created_at DESC);
CREATE INDEX video_renditions_video_height_idx ON media.video_renditions (video_id, height);

-- +goose Down

DROP TABLE media.video_renditions;
DROP TABLE media.video_assets;

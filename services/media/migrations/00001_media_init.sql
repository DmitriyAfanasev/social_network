-- +goose Up

CREATE SCHEMA IF NOT EXISTS media;

CREATE TABLE media.media (
    id uuid PRIMARY KEY,
    object_key varchar(512) NOT NULL UNIQUE,
    bucket varchar(255) NOT NULL,
    original_filename varchar(255) NOT NULL,
    content_type varchar(255),
    media_type varchar(32) NOT NULL,
    size bigint NOT NULL CHECK (size > 0),
    checksum varchar(64),
    width integer,
    height integer,
    duration double precision,
    uploaded_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX media_uploaded_by_created_idx ON media.media (uploaded_by, created_at DESC);
CREATE INDEX media_type_created_idx ON media.media (media_type, created_at DESC);
CREATE INDEX media_active_checksum_idx ON media.media (checksum) WHERE deleted_at IS NULL;

-- +goose Down

DROP TABLE media.media;

-- +goose Up

CREATE TABLE media.video_albums (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    title varchar(80) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX video_albums_user_created_idx
    ON media.video_albums (user_id, created_at DESC, id DESC);

ALTER TABLE media.video_assets
    ADD COLUMN album_id uuid REFERENCES media.video_albums(id) ON DELETE SET NULL;

CREATE INDEX video_assets_album_created_idx
    ON media.video_assets (album_id, created_at DESC, id DESC);

CREATE TABLE media.video_likes (
    id uuid PRIMARY KEY,
    video_id uuid NOT NULL REFERENCES media.video_assets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_id, user_id)
);

CREATE INDEX video_likes_user_idx ON media.video_likes (user_id, created_at DESC);
CREATE INDEX video_likes_video_idx ON media.video_likes (video_id);

CREATE TABLE media.video_favorites (
    id uuid PRIMARY KEY,
    video_id uuid NOT NULL REFERENCES media.video_assets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_id, user_id)
);

CREATE INDEX video_favorites_user_idx ON media.video_favorites (user_id, created_at DESC);
CREATE INDEX video_favorites_video_idx ON media.video_favorites (video_id);

CREATE TABLE media.video_bookmarks (
    id uuid PRIMARY KEY,
    video_id uuid NOT NULL REFERENCES media.video_assets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_id, user_id)
);

CREATE INDEX video_bookmarks_user_idx ON media.video_bookmarks (user_id, created_at DESC);
CREATE INDEX video_bookmarks_video_idx ON media.video_bookmarks (video_id);

CREATE TABLE media.video_views (
    id uuid PRIMARY KEY,
    video_id uuid NOT NULL REFERENCES media.video_assets(id) ON DELETE CASCADE,
    user_id uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX video_views_video_idx ON media.video_views (video_id, created_at DESC);
CREATE INDEX video_views_user_idx ON media.video_views (user_id, created_at DESC);
CREATE UNIQUE INDEX video_views_video_user_unique_idx
    ON media.video_views (video_id, user_id)
    WHERE user_id IS NOT NULL;

-- +goose Down

DROP TABLE media.video_views;
DROP TABLE media.video_bookmarks;
DROP TABLE media.video_favorites;
DROP TABLE media.video_likes;
DROP INDEX media.video_assets_album_created_idx;
ALTER TABLE media.video_assets DROP COLUMN album_id;
DROP TABLE media.video_albums;

-- +goose Up

ALTER TABLE profiles.profiles
    ADD COLUMN avatar_url text NOT NULL DEFAULT '/media/default-avatar';

CREATE TABLE profiles.profile_photo_albums (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    title varchar(80) NOT NULL CHECK (length(btrim(title)) > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX profile_photo_albums_user_created_idx
    ON profiles.profile_photo_albums (user_id, created_at DESC);

CREATE TABLE profiles.profile_photos (
    id uuid PRIMARY KEY,
    album_id uuid NOT NULL REFERENCES profiles.profile_photo_albums(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    media_id uuid NOT NULL,
    caption text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX profile_photos_album_media_unique_idx
    ON profiles.profile_photos (album_id, media_id);
CREATE INDEX profile_photos_user_created_idx
    ON profiles.profile_photos (user_id, created_at DESC);

CREATE TABLE profiles.profile_avatars (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    media_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX profile_avatars_user_media_unique_idx
    ON profiles.profile_avatars (user_id, media_id);
CREATE INDEX profile_avatars_user_created_idx
    ON profiles.profile_avatars (user_id, created_at DESC);

-- +goose Down

DROP TABLE profiles.profile_avatars;
DROP TABLE profiles.profile_photos;
DROP TABLE profiles.profile_photo_albums;
ALTER TABLE profiles.profiles DROP COLUMN avatar_url;

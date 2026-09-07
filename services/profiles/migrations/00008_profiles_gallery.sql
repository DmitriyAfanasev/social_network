-- +goose Up
ALTER TABLE profiles.profile_photo_albums
    ALTER COLUMN title TYPE varchar(128),
    ADD COLUMN description varchar(512) NOT NULL DEFAULT '',
    ADD COLUMN visibility text NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    ADD COLUMN comment_policy text NOT NULL DEFAULT 'public' CHECK (comment_policy IN ('public', 'private', 'nobody'));
ALTER TABLE profiles.profile_photos
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN archived boolean NOT NULL DEFAULT false,
    ADD COLUMN latitude double precision,
    ADD COLUMN longitude double precision,
    ADD CONSTRAINT profile_photo_coordinates CHECK (
        (latitude IS NULL AND longitude IS NULL) OR
        (latitude IS NOT NULL AND longitude IS NOT NULL AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180)
    );
CREATE TABLE profiles.photo_comments (
    id uuid PRIMARY KEY,
    photo_id uuid NOT NULL REFERENCES profiles.profile_photos(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    body varchar(2000) NOT NULL CHECK (length(btrim(body)) > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX photo_comments_photo_created_idx ON profiles.photo_comments(photo_id, created_at, id);
CREATE INDEX profile_photos_media_idx ON profiles.profile_photos(media_id);

-- +goose Down
DROP TABLE profiles.photo_comments;
DROP INDEX profiles.profile_photos_media_idx;
ALTER TABLE profiles.profile_photos DROP CONSTRAINT profile_photo_coordinates,
    DROP COLUMN archived, DROP COLUMN latitude, DROP COLUMN longitude, DROP COLUMN deleted_at;
ALTER TABLE profiles.profile_photo_albums DROP COLUMN description,
    DROP COLUMN visibility, DROP COLUMN comment_policy;
-- Длинные названия сохраняются при откате; исходный varchar(80) терял бы данные.

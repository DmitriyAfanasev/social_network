-- +goose Up

ALTER TABLE profiles.profiles
    ADD COLUMN IF NOT EXISTS profile_details jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS profile_privacy jsonb NOT NULL DEFAULT '{
        "profile_visibility": "everyone",
        "friend_request_policy": "everyone",
        "message_policy": "everyone",
        "phone_visibility": "everyone",
        "birth_date_visibility": "everyone",
        "gender_visibility": "everyone",
        "location_visibility": "everyone",
        "status_visibility": "everyone",
        "friends_visibility": "friends",
        "posts_visibility": "everyone",
        "music_visibility": "friends_of_friends"
    }'::jsonb;

-- +goose Down

ALTER TABLE profiles.profiles
    DROP COLUMN IF EXISTS profile_details,
    DROP COLUMN IF EXISTS profile_privacy;

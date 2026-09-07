-- +goose Up

ALTER TABLE profiles.profiles
    DROP COLUMN display_name;

-- +goose Down

ALTER TABLE profiles.profiles
    ADD COLUMN display_name varchar(80) NOT NULL DEFAULT '';

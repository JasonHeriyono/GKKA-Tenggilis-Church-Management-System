-- +goose Up
ALTER TABLE user_keahlian
    ADD COLUMN sumber ENUM('ADMIN', 'TRAINING') NOT NULL DEFAULT 'ADMIN' AFTER id_keahlian;

-- +goose Down
ALTER TABLE user_keahlian
    DROP COLUMN sumber;

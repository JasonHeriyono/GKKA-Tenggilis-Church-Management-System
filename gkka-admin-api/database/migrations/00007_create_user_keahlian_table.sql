-- +goose Up
CREATE TABLE user_keahlian (
    id_user_keahlian INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_user INT UNSIGNED NOT NULL,
    id_keahlian INT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_user_keahlian (id_user, id_keahlian),
    CONSTRAINT fk_user_keahlian_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE CASCADE,
    CONSTRAINT fk_user_keahlian_keahlian FOREIGN KEY (id_keahlian) REFERENCES keahlian(id_keahlian) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS user_keahlian;

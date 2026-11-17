-- +goose Up
CREATE TABLE user_komunitas (
    id_user_komunitas INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_user INT UNSIGNED NOT NULL,
    id_komunitas INT UNSIGNED NOT NULL,
    is_ketua BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_user_komunitas (id_user, id_komunitas),
    CONSTRAINT fk_user_komunitas_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE CASCADE,
    CONSTRAINT fk_user_komunitas_komunitas FOREIGN KEY (id_komunitas) REFERENCES komunitas(id_komunitas) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS user_komunitas;

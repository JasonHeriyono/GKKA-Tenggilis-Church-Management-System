-- +goose Up
CREATE TABLE keahlian (
    id_keahlian INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    nama_keahlian VARCHAR(100) NOT NULL,
    is_sistem BOOLEAN NOT NULL DEFAULT FALSE, -- TRUE untuk Usher
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_keahlian_nama (nama_keahlian)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS keahlian;

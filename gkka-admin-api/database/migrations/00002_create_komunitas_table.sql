-- +goose Up
CREATE TABLE komunitas (
    id_komunitas INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    nama_komunitas VARCHAR(100) NOT NULL,
    deskripsi TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_komunitas_nama (nama_komunitas)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS komunitas;

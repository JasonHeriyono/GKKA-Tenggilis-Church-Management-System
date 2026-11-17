-- +goose Up
CREATE TABLE opsi_pertanyaan (
    id_opsi INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_pertanyaan INT UNSIGNED NOT NULL,
    label VARCHAR(255) NOT NULL,
    urutan INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_opsi_pertanyaan (id_pertanyaan, label),
    CONSTRAINT fk_opsi_pertanyaan FOREIGN KEY (id_pertanyaan) REFERENCES pertanyaan_evaluasi(id_pertanyaan) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS opsi_pertanyaan;

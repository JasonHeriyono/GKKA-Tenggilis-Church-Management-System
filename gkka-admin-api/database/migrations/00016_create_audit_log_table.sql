-- +goose Up
CREATE TABLE audit_log (
    id_log BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_user INT UNSIGNED NULL, -- aktor
    aksi VARCHAR(20) NOT NULL, -- CREATE/UPDATE/DELETE/RESTORE/IMPORT/LOGIN_GAGAL
    nama_tabel VARCHAR(64) NOT NULL,
    id_data INT UNSIGNED NULL,
    data_lama JSON NULL,
    data_baru JSON NULL,
    ip_address VARCHAR(45) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, -- tanpa updated_at/deleted_at
    KEY idx_audit_tabel_data (nama_tabel, id_data),
    CONSTRAINT fk_audit_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS audit_log;

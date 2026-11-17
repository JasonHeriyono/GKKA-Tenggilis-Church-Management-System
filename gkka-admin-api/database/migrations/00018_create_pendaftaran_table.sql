-- +goose Up
CREATE TABLE pendaftaran (
    id_pendaftaran INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_user INT UNSIGNED NOT NULL, -- akun yang mengajukan verifikasi
    data_diajukan JSON NOT NULL,
    status ENUM('MENUNGGU', 'DISETUJUI', 'DITOLAK') NOT NULL DEFAULT 'MENUNGGU',
    id_user_cocok INT UNSIGNED NULL, -- data jemaat lama yang sama (fitur gabung data, belum dipakai)
    diperiksa_oleh INT UNSIGNED NULL, -- Sekretariat / Majelis
    diperiksa_pada DATETIME NULL,
    catatan_admin TEXT NULL, -- alasan jika ditolak
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    INDEX idx_pendaftaran_status (status, created_at),
    INDEX idx_pendaftaran_user (id_user),
    CONSTRAINT fk_pendaftaran_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE CASCADE,
    CONSTRAINT fk_pendaftaran_user_cocok FOREIGN KEY (id_user_cocok) REFERENCES `user`(id_user) ON DELETE SET NULL,
    CONSTRAINT fk_pendaftaran_pemeriksa FOREIGN KEY (diperiksa_oleh) REFERENCES `user`(id_user) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS pendaftaran;

-- +goose Up
CREATE TABLE absensi (
    id_absensi INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_kegiatan INT UNSIGNED NOT NULL,
    id_user INT UNSIGNED NOT NULL,
    waktu_absen DATETIME NOT NULL,
    metode VARCHAR(10) NOT NULL, -- TAP / MANUAL
    pencatat_id INT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_absensi (id_kegiatan, id_user),
    CONSTRAINT fk_absensi_kegiatan FOREIGN KEY (id_kegiatan) REFERENCES kegiatan(id_kegiatan) ON DELETE CASCADE,
    CONSTRAINT fk_absensi_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE RESTRICT,
    CONSTRAINT fk_absensi_pencatat FOREIGN KEY (pencatat_id) REFERENCES `user`(id_user) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS absensi;

-- +goose Up
CREATE TABLE jadwal_pelayanan (
    id_penjadwalan INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_kegiatan INT UNSIGNED NOT NULL,
    id_user INT UNSIGNED NOT NULL,
    id_keahlian INT UNSIGNED NOT NULL,
    status_konfirmasi VARCHAR(20) NOT NULL DEFAULT 'Confirmed',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_jadwal_pelayanan (id_kegiatan, id_user, id_keahlian),
    CONSTRAINT fk_jadwal_kegiatan FOREIGN KEY (id_kegiatan) REFERENCES kegiatan(id_kegiatan) ON DELETE CASCADE,
    CONSTRAINT fk_jadwal_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE RESTRICT,
    CONSTRAINT fk_jadwal_keahlian FOREIGN KEY (id_keahlian) REFERENCES keahlian(id_keahlian) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS jadwal_pelayanan;

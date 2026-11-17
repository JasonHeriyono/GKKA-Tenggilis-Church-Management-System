-- +goose Up
CREATE TABLE kegiatan (
    id_kegiatan INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    nama_kegiatan VARCHAR(150) NOT NULL,
    tanggal DATE NOT NULL,
    waktu_mulai TIME NOT NULL,
    waktu_selesai TIME NULL,
    pembicara VARCHAR(100) NULL,
    tema VARCHAR(255) NULL,
    nats_alkitab VARCHAR(100) NULL,
    lokasi VARCHAR(150) NULL,
    deskripsi TEXT NULL,
    id_komunitas INT UNSIGNED NULL,
    perlu_evaluasi BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_kegiatan (nama_kegiatan, tanggal, waktu_mulai),
    KEY idx_kegiatan_tanggal (tanggal),
    CONSTRAINT fk_kegiatan_komunitas FOREIGN KEY (id_komunitas) REFERENCES komunitas(id_komunitas) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS kegiatan;

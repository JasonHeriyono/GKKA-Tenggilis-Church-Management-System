-- +goose Up
CREATE TABLE kuesioner_evaluasi (
    id_kuesioner INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    jenis_kuesioner VARCHAR(10) NOT NULL, -- MINGGUAN / TAHUNAN
    id_kegiatan INT UNSIGNED NULL, -- wajib untuk MINGGUAN
    tahun INT NULL, -- wajib untuk TAHUNAN
    judul_form VARCHAR(255) NOT NULL,
    tema VARCHAR(255) NULL, -- snapshot
    pembicara VARCHAR(100) NULL, -- snapshot
    dibuka_pada DATETIME NULL,
    ditutup_pada DATETIME NULL,
    kode_akses VARCHAR(8) NOT NULL, -- dibuat backend, 6 karakter, contoh K7MP2X
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_kuesioner_kegiatan (id_kegiatan),
    UNIQUE KEY uk_kuesioner_jenis_tahun (jenis_kuesioner, tahun),
    UNIQUE KEY uk_kuesioner_kode (kode_akses),
    CONSTRAINT fk_kuesioner_kegiatan FOREIGN KEY (id_kegiatan) REFERENCES kegiatan(id_kegiatan) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS kuesioner_evaluasi;

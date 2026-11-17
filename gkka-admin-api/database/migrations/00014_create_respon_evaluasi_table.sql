-- +goose Up
CREATE TABLE respon_evaluasi (
    id_respon INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_kuesioner INT UNSIGNED NULL, -- NULL jika kode tidak cocok dengan kuesioner mana pun
    id_user INT UNSIGNED NULL, -- NULL = anonim
    kode_dimasukkan VARCHAR(8) NULL, -- kode yang diketik jemaat di form
    status_respon VARCHAR(10) NOT NULL DEFAULT 'VALID', -- VALID / INVALID
    alasan_invalid VARCHAR(20) NULL, -- KODE_SALAH / LEWAT_WAKTU / BELUM_DIBUKA
    waktu_submit DATETIME NOT NULL,
    google_response_id VARCHAR(150) NOT NULL, -- mencegah respon yang sama tersimpan dua kali
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_respon_kuesioner_user (id_kuesioner, id_user),
    UNIQUE KEY uk_respon_google (google_response_id),
    CONSTRAINT fk_respon_kuesioner FOREIGN KEY (id_kuesioner) REFERENCES kuesioner_evaluasi(id_kuesioner) ON DELETE CASCADE,
    CONSTRAINT fk_respon_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS respon_evaluasi;

-- +goose Up
CREATE TABLE jawaban_evaluasi (
    id_jawaban INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    id_respon INT UNSIGNED NOT NULL,
    id_pertanyaan INT UNSIGNED NOT NULL,
    id_opsi INT UNSIGNED NULL, -- untuk tipe pilihan; CENTANG = beberapa baris
    nilai_rating TINYINT NULL, -- 1-5
    nilai_teks TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_jawaban (id_respon, id_pertanyaan, id_opsi),
    CONSTRAINT fk_jawaban_respon FOREIGN KEY (id_respon) REFERENCES respon_evaluasi(id_respon) ON DELETE CASCADE,
    CONSTRAINT fk_jawaban_pertanyaan FOREIGN KEY (id_pertanyaan) REFERENCES pertanyaan_evaluasi(id_pertanyaan) ON DELETE RESTRICT,
    CONSTRAINT fk_jawaban_opsi FOREIGN KEY (id_opsi) REFERENCES opsi_pertanyaan(id_opsi) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS jawaban_evaluasi;

-- +goose Up
CREATE TABLE pertanyaan_evaluasi (
    id_pertanyaan INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    jenis_kuesioner VARCHAR(10) NOT NULL, -- MINGGUAN / TAHUNAN
    teks_pertanyaan TEXT NOT NULL,
    tipe_jawaban VARCHAR(20) NOT NULL, -- RATING/TEKS_SINGKAT/PILIHAN_GANDA/CENTANG/DROPDOWN/PARAGRAF
    urutan INT NOT NULL DEFAULT 0,
    wajib BOOLEAN NOT NULL DEFAULT FALSE,
    is_aktif BOOLEAN NOT NULL DEFAULT TRUE,
    google_item_id VARCHAR(100) NULL,
    google_question_id VARCHAR(100) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_pertanyaan_google_question (google_question_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS pertanyaan_evaluasi;

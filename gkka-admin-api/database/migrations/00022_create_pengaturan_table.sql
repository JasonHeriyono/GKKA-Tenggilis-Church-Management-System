-- +goose Up
-- pengaturan aplikasi yang bisa diubah admin, format kunci-nilai
-- contoh kunci: evaluasi.lama_buka_hari, header_form.MINGGUAN.judul
CREATE TABLE pengaturan (
    kunci VARCHAR(100) NOT NULL PRIMARY KEY,
    nilai TEXT NOT NULL,
    deskripsi VARCHAR(255) NULL, -- penjelasan untuk admin
    diubah_oleh INT UNSIGNED NULL, -- admin terakhir yang mengubah
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_pengaturan_user FOREIGN KEY (diubah_oleh) REFERENCES `user`(id_user) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS pengaturan;

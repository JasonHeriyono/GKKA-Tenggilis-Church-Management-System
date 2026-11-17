-- +goose Up
CREATE TABLE `user` (
    id_user INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    kode_jemaat VARCHAR(20) NULL, -- diberikan saat terverifikasi
    nama VARCHAR(100) NOT NULL,
    email VARCHAR(100) NULL, -- login pakai email atau no HP
    no_telepon VARCHAR(20) NULL,
    password_hash VARCHAR(255) NULL, -- NULL = tidak punya akun login (mis. anak)
    id_role INT UNSIGNED NOT NULL,
    status_verifikasi VARCHAR(25) NOT NULL DEFAULT 'BELUM_TERVERIFIKASI', -- BELUM_TERVERIFIKASI / MENUNGGU_VERIFIKASI / TERVERIFIKASI
    persetujuan_data_pada DATETIME NULL, -- consent UU PDP saat signup
    is_aktif BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_user_kode_jemaat (kode_jemaat),
    UNIQUE KEY uk_user_email (email),
    UNIQUE KEY uk_user_no_telepon (no_telepon),
    CONSTRAINT fk_user_role FOREIGN KEY (id_role) REFERENCES role(id_role) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `user`;

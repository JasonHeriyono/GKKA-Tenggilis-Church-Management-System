-- +goose Up
CREATE TABLE identitas_user (
    id_user INT UNSIGNED PRIMARY KEY,
    nik VARCHAR(16) NULL, -- 16 digit, hanya Super Admin
    nama_panggilan VARCHAR(50) NULL,
    jenis_kelamin ENUM('L', 'P') NULL,
    tempat_lahir VARCHAR(100) NULL,
    tanggal_lahir DATE NULL,
    alamat_ktp TEXT NULL,
    rt_rw VARCHAR(10) NULL,
    kelurahan VARCHAR(100) NULL,
    kecamatan VARCHAR(100) NULL,
    kota VARCHAR(100) NULL,
    provinsi VARCHAR(100) NULL,
    alamat_domisili TEXT NULL, -- jika beda dari KTP
    kewarganegaraan ENUM('WNI', 'WNA') NULL,
    status_pernikahan ENUM('BELUM_KAWIN', 'KAWIN', 'CERAI_HIDUP', 'CERAI_MATI') NULL,
    tanggal_pernikahan DATE NULL,
    pendidikan_terakhir ENUM('SD', 'SMP', 'SMA', 'D3', 'S1', 'S2', 'S3') NULL,
    pekerjaan VARCHAR(100) NULL,
    nama_ayah VARCHAR(100) NULL,
    nama_ibu VARCHAR(100) NULL,
    status_gereja ENUM('JEMAAT_GKKA', 'SIMPATISAN', 'TAMU') NULL,
    sudah_baptis BOOLEAN NOT NULL DEFAULT FALSE,
    tanggal_baptis DATE NULL,
    gereja_baptis VARCHAR(150) NULL,
    sudah_sidi BOOLEAN NOT NULL DEFAULT FALSE,
    tanggal_sidi DATE NULL,
    gereja_sidi VARCHAR(150) NULL,
    gereja_asal VARCHAR(150) NULL, -- jika pindahan (atestasi)
    tanggal_bergabung DATE NULL,
    tanggal_keluar DATE NULL,
    alasan_keluar ENUM('PINDAH', 'ATESTASI_KELUAR', 'MENINGGAL') NULL,
    data_tambahan JSON NULL, -- isian lain yang tidak difilter
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_identitas_nik (nik),
    CONSTRAINT fk_identitas_user FOREIGN KEY (id_user) REFERENCES `user`(id_user) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS identitas_user;

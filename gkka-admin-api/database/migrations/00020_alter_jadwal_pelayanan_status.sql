-- +goose Up
-- ERD R2: status_konfirmasi = MENUNGGU / TERKONFIRMASI / DITOLAK, default TERKONFIRMASI.
-- Sebelumnya default 'Confirmed'; baris lama ikut dikonversi.
ALTER TABLE jadwal_pelayanan
    MODIFY status_konfirmasi VARCHAR(20) NOT NULL DEFAULT 'TERKONFIRMASI';
UPDATE jadwal_pelayanan SET status_konfirmasi = 'TERKONFIRMASI' WHERE status_konfirmasi = 'Confirmed';

-- +goose Down
UPDATE jadwal_pelayanan SET status_konfirmasi = 'Confirmed' WHERE status_konfirmasi = 'TERKONFIRMASI';
ALTER TABLE jadwal_pelayanan
    MODIFY status_konfirmasi VARCHAR(20) NOT NULL DEFAULT 'Confirmed';

# SIM-G Backend

Backend REST API untuk **SIM-G (Sistem Informasi Manajemen Gereja)**, Gereja GKKA Tenggilis, Surabaya.
Proyek Pengabdian kepada Masyarakat (PkM) Kelompok 22, Informatika Universitas Kristen Petra, 2026.

Satu API ini dipakai bersama oleh:

- **Web Admin** (React/Next.js): untuk sekretariat, Super Admin, dan Majelis
- **Aplikasi Mobile Jemaat** (Flutter): untuk jemaat dan usher

---

## Daftar Isi

- [Fitur](#fitur)
- [Tech Stack](#tech-stack)
- [Prasyarat](#prasyarat)
- [Cara Menjalankan](#cara-menjalankan)
- [Konfigurasi Environment](#konfigurasi-environment)
- [Struktur Folder](#struktur-folder)
- [Library](#library)
- [Konvensi](#konvensi)
- [Tim](#tim)

---

## Fitur

### Modul Inti (MVP)

| Fitur                                | Modul               | Ringkasan                                                                                                                                         |
| ------------------------------------ | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Hak akses berbasis peran (RBAC)      | `auth`, `user`      | Role tetap: **Jemaat**, **Majelis** (hanya lihat laporan), **Super Admin**. **Usher** adalah hak sementara bagi jemaat yang dijadwalkan bertugas. |
| Group & Skill Tagging                | `group`, `skill`    | Mengelompokkan jemaat per komunitas (Pemuda, Sekolah Minggu, Wilayah) dan keahlian (Pemusik, Usher, Singer).                                      |
| Smart Batch Import & Clash Detection | `schedule`          | Upload jadwal dari template Excel (preview → commit, tanpa duplikat), dengan peringatan bentrok waktu atau skill.                                 |
| Sistem Absensi Serbaguna             | `attendance`        | Tap-to-absen oleh usher (Mobile) dan input manual (Web Admin).                                                                                    |
| Kuesioner Evaluasi & Dashboard       | `evaluation`        | Satu form kuesioner. Header (tema, tanggal, pembicara) terisi otomatis dari jadwal, tanpa cron. Ada dashboard untuk Majelis dan export Excel.     |
| Activity Logs & Audit Trail          | `activity`, `audit` | Riwayat keikutsertaan jemaat dan catatan perubahan data oleh pengurus.                                                                            |

### Pengembangan Lanjutan (opsional)

Push notification / broadcast, export PDF, audit trail lanjutan, dan Dynamic RBAC.

---

## Tech Stack

| Kebutuhan        | Teknologi                            |
| ---------------- | ------------------------------------ |
| Bahasa           | Go 1.26+                             |
| HTTP framework   | Gin                                  |
| ORM & database   | GORM + MySQL 8                       |
| Migrasi database | goose                                |
| Auth             | JWT (access + refresh token), bcrypt |
| Dokumentasi API  | Swagger (swaggo)                     |
| Kuesioner        | Google Forms API (service account)   |
| Deployment       | Docker + GitHub Actions → VPS        |

---

## Prasyarat

Pastikan sudah ter-install:

- [Go](https://go.dev/dl/) **1.26 atau lebih baru**: cek dengan `go version`
- [MySQL](https://dev.mysql.com/downloads/) **8.x**, atau XAMPP/Laragon yang menyertakan MySQL
- [Git](https://git-scm.com/)

---

## Cara Menjalankan

### 1. Clone repository

```bash
git clone <url-repository>
cd sim-g-gkka-backend
```

### 2. Buat database

Masuk ke MySQL, lalu buat database kosong. Nama database ini nanti dimasukkan ke `DB_NAME` di file `.env`.

### 3. Siapkan file `.env`

Salin `.env.example`, lalu sesuaikan isinya (terutama `DB_PASSWORD` dan `JWT_SECRET`):

```bash
cp .env.example .env
```

> Di Windows (PowerShell): `Copy-Item .env.example .env`

### 4. Download library

```bash
go mod download
```

Perintah ini hanya mengunduh library yang tercatat di `go.mod`, tanpa mengubah isi `go.mod`.

> **Jangan jalankan `go mod tidy` hanya untuk setup.** `go mod tidy` menghapus library yang belum di-`import` di file `.go` mana pun. Jalankan `go mod tidy` hanya setelah kamu menambah atau menghapus `import` di kode, lalu commit perubahan `go.mod` dan `go.sum`.

### 5. Buat tabel (migrasi)

```bash
go tool goose up
```

goose membaca `GOOSE_DRIVER`, `GOOSE_DBSTRING`, dan `GOOSE_MIGRATION_DIR` dari `.env`, lalu menjalankan file SQL di `database/migrations/` secara berurutan.

### 6. Isi data awal (seed)

Membuat role default dan akun Super Admin pertama (isi dulu `SEED_ADMIN_*` di `.env`):

```bash
go run . seed
```

### 7. Jalankan aplikasi

```bash
go run .
```

Server berjalan di `http://localhost:8080` (sesuai `APP_PORT`). Semua endpoint diawali `/gkki/api`, contoh: `http://localhost:8080/gkki/api/health`.

### Perintah lain

| Perintah               | Fungsi                                                                    |
| ---------------------- | ------------------------------------------------------------------------- |
| `go test ./...`        | Menjalankan semua unit test                                               |
| `go tool air`          | Menjalankan server dengan hot reload (otomatis restart saat file berubah) |
| `go tool swag init`    | Generate dokumentasi Swagger ke folder `docs/`                            |
| `go tool goose status` | Melihat migrasi mana yang sudah / belum dijalankan                        |
| `go tool goose down`   | Membatalkan migrasi terakhir                                              |

> Swagger UI (`/swagger/index.html`) belum bisa dibuka: route `gin-swagger` belum dipasang di `routes/routes.go`.

---

## Konfigurasi Environment

| Variabel               | Contoh                  | Keterangan                                                                       |
| ---------------------- | ----------------------- | -------------------------------------------------------------------------------- |
| `APP_ENV`              | `development`           | `development` / `production`                                                     |
| `APP_PORT`             | `8080`                  | Port server                                                                      |
| `APP_TIMEZONE`         | `Asia/Jakarta`          | Zona waktu aplikasi (WIB), dipakai untuk jadwal dan absensi                      |
| `DB_HOST`              | `localhost`             | Host MySQL                                                                       |
| `DB_PORT`              | `3306`                  | Port MySQL                                                                       |
| `DB_USER`              | `root`                  | User MySQL                                                                       |
| `DB_PASSWORD`          | —                       | Password MySQL                                                                   |
| `DB_NAME`              | `sim_g`                 | Nama database                                                                    |
| `JWT_SECRET`           | —                       | **Wajib** di semua environment, string acak min. 32 karakter (`openssl rand -hex 32`) |
| `JWT_ACCESS_TTL`       | `15m`                   | Masa berlaku access token                                                        |
| `JWT_REFRESH_TTL`      | `168h`                  | Masa berlaku refresh token (7 hari)                                              |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Domain Web Admin yang diizinkan mengakses API                                    |
| `TRUSTED_PROXIES`      | —                       | IP reverse proxy (Nginx) di VPS. Kosongkan jika server diakses langsung          |
| `GOOSE_DRIVER`         | `mysql`                 | Driver database untuk goose                                                      |
| `GOOSE_DBSTRING`       | lihat `.env.example`    | Koneksi database untuk goose (otomatis memakai nilai `DB_*`)                     |
| `GOOSE_MIGRATION_DIR`  | `database/migrations`   | Folder file migrasi                                                              |
| `SEED_ADMIN_KODE`      | `ADMIN001`              | Kode jemaat Super Admin pertama (dipakai `go run . seed`)                        |
| `SEED_ADMIN_NAMA`      | `Super Admin`           | Nama Super Admin pertama                                                         |
| `SEED_ADMIN_EMAIL`     | `admin@example.com`     | Email login Super Admin pertama                                                  |
| `SEED_ADMIN_PASSWORD`  | —                       | Password Super Admin pertama, min. 12 karakter. Hapus dari `.env` setelah login dan ganti password |
| `GOOGLE_CREDENTIALS_FILE` | `service-account.json` | File JSON key service account Google (sudah di `.gitignore`, jangan di-commit) |
| `GOOGLE_FORM_ID_MINGGUAN` | —                    | ID Google Form kuesioner mingguan (dari URL `/forms/d/<ID>/edit`). Kosong = fitur Google mati |
| `GOOGLE_FORM_ID_TAHUNAN`  | —                    | ID Google Form kuesioner tahunan |

> File `.env` berisi data rahasia dan **tidak boleh di-commit** (sudah masuk `.gitignore`).

> Saat `APP_ENV=production`, server **menolak start** jika: `DB_PASSWORD` kosong, `DB_USER=root`, atau `CORS_ALLOWED_ORIGINS` berisi `*` / `localhost` / `127.0.0.1`. Di VPS, set `APP_ENV=production` secara eksplisit.

---

## Struktur Folder

```
sim-g-backend/
├── main.go              # titik mulai aplikasi: go run . | go run . seed
├── config/              # membaca .env → struct config
├── database/            # koneksi, transaksi, seeder, migrations/
├── routes/              # daftar semua endpoint + group per role
├── middleware/          # auth, role, recovery, cors, logger, request_id, rate_limit
├── modules/             # 1 folder = 1 fitur
│   ├── auth/            # login, refresh token, logout
│   ├── user/            # profil jemaat, role
│   ├── group/           # komunitas
│   ├── skill/           # skill tagging
│   ├── schedule/        # jadwal, import Excel, clash detection, usher bertugas
│   ├── attendance/      # absensi
│   ├── evaluation/      # kuesioner, dashboard, export
│   ├── activity/        # riwayat keikutsertaan jemaat
│   └── audit/           # audit trail
├── models/              # struct tabel database (mengikuti ERD)
├── helpers/             # response, apperror, pagination, currentuser, excel
├── docs/                # hasil generate Swagger (jangan diedit manual)
└── templates/           # template Excel import jadwal
```

Setiap folder di `modules/` memakai pola yang sama:

| File            | Tugas                                                                  |
| --------------- | ---------------------------------------------------------------------- |
| `handler.go`    | Menerima request, validasi input, memanggil service, mengirim response |
| `service.go`    | Logika bisnis (tidak tahu soal HTTP)                                   |
| `repository.go` | Query ke database lewat GORM                                           |
| `dto.go`        | Bentuk data request/response API (acuan tim Web & Mobile)              |

Alur request: **routes → middleware → handler → service → repository → database**

Penjelasan detail setiap folder & file ada di [`STRUKTUR_FOLDER.md`](STRUKTUR_FOLDER.md) (mengacu pada `SIM-G_Backend_Library_Struktur_Folder_v2.pdf`).

---

## Library

### Core

| Library                                                                                             | Kegunaan                           |
| --------------------------------------------------------------------------------------------------- | ---------------------------------- |
| [gin-gonic/gin](https://github.com/gin-gonic/gin)                                                   | HTTP router & middleware           |
| [gorm.io/gorm](https://gorm.io) + `gorm.io/driver/mysql`                                            | ORM & driver MySQL                 |
| [pressly/goose](https://github.com/pressly/goose)                                                   | Migrasi database berbasis file SQL |
| [joho/godotenv](https://github.com/joho/godotenv) + [caarlos0/env](https://github.com/caarlos0/env) | Membaca `.env` ke struct config    |
| [go-playground/validator](https://github.com/go-playground/validator)                               | Validasi input request             |
| [google/uuid](https://github.com/google/uuid)                                                       | ID unik                            |
| `log/slog` (bawaan Go)                                                                              | Logging terstruktur                |

### Auth & Keamanan

| Library                                                 | Kegunaan                         |
| ------------------------------------------------------- | -------------------------------- |
| [golang-jwt/jwt](https://github.com/golang-jwt/jwt)     | Access & refresh token           |
| `golang.org/x/crypto/bcrypt`                            | Hash password                    |
| [gin-contrib/cors](https://github.com/gin-contrib/cors) | Izin akses dari domain Web Admin |
| `golang.org/x/time/rate`                                | Rate limit percobaan login       |

### Fitur

| Library                                                       | Kegunaan                             |
| ------------------------------------------------------------- | ------------------------------------ |
| [xuri/excelize](https://github.com/xuri/excelize)             | Import jadwal & export laporan Excel |
| [swaggo/swag](https://github.com/swaggo/swag) + `gin-swagger` | Dokumentasi API (Swagger UI)         |
| [google.golang.org/api/forms](https://pkg.go.dev/google.golang.org/api/forms/v1) + `golang.org/x/oauth2` | Sinkron pertanyaan & header kuesioner ke Google Form, membaca jawaban |

### Development & Testing

| Library                                                       | Kegunaan                            |
| ------------------------------------------------------------- | ----------------------------------- |
| [stretchr/testify](https://github.com/stretchr/testify)       | Assertion & mock untuk unit test    |
| [DATA-DOG/go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) | Mock database untuk test repository |
| [air-verse/air](https://github.com/air-verse/air)             | Hot reload saat development         |
| [golangci-lint](https://golangci-lint.run)                    | Linter (dijalankan di CI)           |

---

## Konvensi

### Penamaan

- **Folder / package**: huruf kecil, satu kata, tanpa underscore (`schedule`, `currentuser`)
- **File**: snake_case yang menjelaskan isinya (`clash_detector.go`, `excel_importer.go`)
- **File test**: nama file + `_test.go` (`clash_detector_test.go`)

### Format Commit

| Prefix      | Dipakai untuk                            |
| ----------- | ---------------------------------------- |
| `feat:`     | Fitur baru                               |
| `fix:`      | Perbaikan bug                            |
| `refactor:` | Perubahan kode tanpa mengubah perilaku   |
| `docs:`     | Dokumentasi                              |
| `test:`     | Menambah atau memperbaiki test           |
| `chore:`    | Konfigurasi, dependency, struktur folder |

Contoh: `feat: tambah endpoint import jadwal dari Excel`

---

## Tim

| Nama                      | Peran                                    |
| ------------------------- | ---------------------------------------- |
| Evan Suryadinata S        | Project Manager & System Architect       |
| Robert Leonardo Sanjaya   | Database Engineer & Requirements Analyst |
| Nicholas Valentino Candra | Backend Developer                        |
| Christofer Jason Hariyono | Frontend Developer & UI/UX               |
| Jose Errando Saputra      | Mobile & Backend Developer               |
| Ariel Wilson Sahli        | DevOps, QA & Training Coordinator        |

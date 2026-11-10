# AGENTS.md — Quevia Coding Agent Operating Manual

Dokumen ini adalah **buku panduan operasional wajib (rulebook)** bagi seluruh AI coding agent yang bekerja di repositori **Quevia**. Setiap agent wajib mematuhi standar arsitektur, pola kode, konvensi Git, dan batasan operasional yang ditetapkan di bawah ini.

---

## 1. Identitas Proyek & Arsitektur Inti

- **Nama Produk**: Quevia (Smart Queue & Appointment Management SaaS)
- **Tipe Proyek**: B2B SaaS Multi-Tenant (Monorepo)
- **Pola Arsitektur**: **Modular Monolith** dua proses runtime (`api` dan `worker`), tanpa arsitektur microservices.
- **Tech Stack**:
  - **Backend**: Golang 1.22+ dengan framework Hertz, sqlc, pgx pool.
  - **Worker**: Golang dengan River (PostgreSQL transactional job engine) dan Outbox Relay.
  - **Frontend**: Next.js 14+ (App Router), TypeScript, Tailwind CSS, shadcn/ui, TanStack Query, Zod.
  - **Database**: PostgreSQL 16+ dengan ekstensi `btree_gist` dan Row-Level Security (RLS).
  - **Cache & Pub/Sub**: Redis 7+ (Rate limiter, referensi cache, session token, real-time pub/sub).
  - **Real-Time Gateway**: WebSockets (`coder/websocket`) multiplexed via Redis.

---

## 2. Peta Dokumen Acuan (Single Source of Truth)

Sebelum membuat atau mengubah kode, rujuk dokumen spesifikasi resmi di repositori ini:
- 📄 `docs/prd.md` — Spesifikasi fungsional, 10 User Stories, dan 21 Aturan Bisnis.
- 🏛️ `docs/architecture.md` — Alur runtime, kontrak REST API OpenAPI 3.0, dan katalog event WebSocket.
- 🗄️ `docs/erd.md` — Skema 26 tabel database, constraint GiST, indeks, dan DDL RLS.
- 🗺️ `IMPLEMENTATION_PLAN.md` — Rencana eksekusi 8 minggu (4 sprint) beserta task checklist & verification gates.
- 📜 `CONTRIBUTING.md` — Panduan Git, alur branching, dan standarisasi commit message.

---

## 3. Tujuh Hukum Arsitektur Quevia (Anti-Halusinasi)

Agent **DILARANG KERAS** melanggar 7 aturan mutlak ini:

1. **Isolasi Multi-Tenant Mutlak (PostgreSQL RLS)**:
   - Seluruh query database yang mengakses tabel ber-`tenant_id` wajib dibuka melalui transaksi helper `WithTx` yang mengeksekusi `set_config('app.current_tenant_id', $tenant_id, true)`.
   - Dilarang hanya mengandalkan filter `WHERE tenant_id = ...` di level aplikasi. Database RLS dengan `FORCE ROW LEVEL SECURITY` adalah penjaga terakhir.
2. **Larangan Ekspos Internal ID**:
   - Kolom `id` (BIGINT internal) hanya digunakan untuk relasi foreign key dan join internal database.
   - Dilarang mengekspos integer `id` ke URL endpoint, parameter query, atau respons payload HTTP. Gunakan selalu `guid` (UUID v4).
3. **Pencegahan Double-Booking Mutlak (GiST Exclusion)**:
   - Double-booking janji temu dicegah langsung di level kernel PostgreSQL melalui constraint:
     `CONSTRAINT ex_appointments_no_overlap EXCLUDE USING gist (staff_id WITH =, tstzrange(start_at, end_at) WITH &&) WHERE (status IN ('CONFIRMED','CHECKED_IN') AND deleted_at IS NULL)`.
   - Pengecekan ketersediaan slot di aplikasi bersifat informatif; exclusion constraint database adalah sumber kebenaran mutlak.
4. **Penerbitan Nomor Antrean Deterministik (Row-Locking Counter)**:
   - Nomor antrean harian dibuat di dalam transaksi dengan mengunci counter:
     `INSERT INTO queue_counters (...) ON CONFLICT (branch_id, service_id, queue_date) DO UPDATE SET last_number = queue_counters.last_number + 1 RETURNING last_number`.
   - Dilarang menggunakan `SELECT MAX(queue_number) + 1` karena rentan *race condition*.
5. **Pemanggilan Antrean Bebas Tabrakan (Call Next)**:
   - Operator memanggil antrean berikutnya menggunakan query:
     `SELECT ... FROM queue_tickets WHERE ... AND status = 'WAITING' ORDER BY priority DESC, COALESCE(checked_in_at, created_at), queue_number LIMIT 1 FOR UPDATE SKIP LOCKED`.
   - Menjamin dua operator loket tidak akan pernah memanggil tiket yang sama.
6. **Notifikasi Asynchronous (No Blocking in HTTP Path)**:
   - Pengiriman email konfirmasi dan pengingat berjenjang (T-24h, T-60m, T-15m) dilarang berada di dalam HTTP request handler.
   - Handler hanya menulis ke tabel `notifications` dan `outbox_events` atau menjadwalkan job di River. Worker yang akan mengirimkannya ke provider SMTP dengan retry exponential backoff.
7. **Idempotensi Operasi Kritis**:
   - Endpoint `POST /appointments`, `POST /queue/walk-in`, dan `POST /check-in` wajib memvalidasi header `Idempotency-Key` menggunakan tabel `idempotency_keys` untuk mencegah eksekusi ganda pada network retry.

---

## 4. Standar Git, Branching, & Commit

### 4.1 Aturan Branching
- `master`: Cabang produksi stabil (`protected`). Hanya menerima merge dari `development` dan `hotfix/*`.
- `development`: Cabang utama integrasi (`protected`). Seluruh task dimulai dengan membuat branch dari sini.
- `feature/*`: Format `feature/{modul}/{sub_modul}` (contoh: `feature/setup/project_scaffolding`, `feature/appointment/booking`).
- Selalu gabungkan (*merge*) kembali ke `development` melalui Merge Request (MR).

### 4.2 Standar Pesan Commit
Format pesan commit:
```text
<type>(<scope>): <deskripsi singkat> (maksimal 80 karakter)

   - Rincian perubahan teknis 1
   - Rincian perubahan teknis 2

[Kategori Layanan / Tag]
```
Kategori `type`: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `style`, `perf`, `build`, `ci`.

Contoh:
```text
feat(appointment): implementasi kalkulasi ketersediaan slot bebas bentrok

   - Mengambil jadwal mingguan staff dan pengecualian libur/cuti
   - Memfilter slot berdasarkan rentang tstzrange janji temu aktif
   - Menambahkan unit test kalkulasi slot

[API]
```

---

## 5. Perintah Pengembang Utama (Quick Commands)

```bash
# Menjalankan backing services lokal (PostgreSQL 16, Redis 7, Mailpit)
make up

# Melihat log container
make logs

# Menghentikan layanan
make down

# Status layanan Docker
docker compose ps

# URL Mailpit Web UI (Pemeriksaan email testing lokal)
http://localhost:8025
```

---

## 6. Protokol Menjalankan Task dari `IMPLEMENTATION_PLAN.md`

Ketika pengguna meminta untuk mengeksekusi suatu task (misal: *"Kerjakan Task 1.1"* atau *"Lanjut ke Task 2.2"*):
1. **Baca Task Terkait**: Buka `IMPLEMENTATION_PLAN.md` dan pelajari deskripsi, ukuran (S/M/L), daftar file terdampak, dan *Acceptance Criteria*.
2. **Buat Feature Branch**:
   ```bash
   git checkout development
   git pull origin development
   git checkout -b feature/{modul}/{nama_task}
   ```
3. **Tulis Kode Berkualitas**:
   - Patuhi 7 Hukum Arsitektur di Bagian 3.
   - Pisahkan layer: handler $\rightarrow$ service/usecase $\rightarrow$ repository (sqlc) $\rightarrow$ model.
   - Jangan letakkan business logic di handler HTTP.
4. **Verifikasi Teknis**:
   - Jalankan perintah verifikasi yang tertulis di task (misal: `go test -v ./...`, `npm run build`).
5. **Commit Standar**:
   - Commit perubahan dengan format standar di Bagian 4.2.
6. **Lapor & Tandai Selesai**:
   - Laporkan hasil pengujian kepada pengguna beserta ringkasan perubahan.

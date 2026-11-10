# System Architecture & API Contract — Quevia

Versi 1.1 · 10 November 2026 · Penulis: Sarah & Fatah · Mengacu pada PRD Quevia v1.2 dan ERD Quevia v1.1

## 1. Ringkasan dan Prinsip

Dokumen ini menjelaskan bagaimana Quevia dibangun: arsitektur sistem, struktur backend dan frontend, integrasi real-time, autentikasi, kontrak API, dan deployment. PRD menjelaskan apa yang dibangun, ERD menjelaskan datanya, dan dokumen ini menjelaskan bagaimana semuanya bekerja bersama. Cakupan MVP adalah website responsif; tidak ada aplikasi mobile.

Prinsip arsitektur:

1. **Modular monolith, dua proses.** Satu codebase menghasilkan proses api dan proses worker; tidak ada microservices.
2. **Database sebagai sumber kebenaran.** Real-time hanyalah proyeksi: event boleh terlambat atau hilang, dan klien selalu dapat sinkron ulang dari snapshot.
3. **Operasi kritis transaksional dan idempotent.** Booking, nomor antrean, call next, dan check-in berjalan dalam transaksi dan aman terhadap request ulang.
4. **Isolasi tenant berlapis.** Middleware menentukan tenant, query difilter di aplikasi, dan PostgreSQL menegakkan row-level security (RLS).
5. **Notifikasi tidak pernah berada di jalur request.** Email dan reminder diproses oleh background worker (River).
6. **Kontrak API lebih dulu.** Spesifikasi OpenAPI menjadi acuan handler, dokumentasi, dan tipe di frontend.

---

## 2. Arsitektur Sistem

### 2.1 Diagram Runtime

```
                    [ Browser / Mobile Client / Display Kiosk ]
                                     │
                     ┌───────────────┴───────────────┐
             HTTPS / REST                      WSS / WebSocket
                     │                               │
                     ▼                               ▼
       ┌──────────────────────────────────────────────────────────┐
       │                   Reverse Proxy (TLS)                    │
       └─────────────────────────────┬────────────────────────────┘
                                     │
                     ┌───────────────┴───────────────┐
                     ▼                               ▼
      ┌─────────────────────────────┐  ┌─────────────────────────────┐
      │   Frontend (Next.js App)    │  │    Backend API (Go Hertz)   │
      └─────────────────────────────┘  │   - REST Endpoint Handlers  │
                                       │   - WebSocket Gateway       │
                                       └──────────────┬──────────────┘
                                                      │
                     ┌────────────────────────────────┼────────────────────────────────┐
                     ▼                                ▼                                ▼
      ┌─────────────────────────────┐  ┌─────────────────────────────┐  ┌─────────────────────────────┐
      │    PostgreSQL (Data Inti)   │  │   Redis (Cache & Pub/Sub)   │  │    Worker (Go + River)      │
      │  - 26 Tabel & RLS Multi-    │  │  - Reference Cache          │  │  - Outbox Event Relay       │
      │    Tenant Policies          │  │  - Rate Limiter Token       │  │  - Email Sender (SMTP)      │
      │  - GiST Exclusion Constraint│  │  - Real-Time Pub/Sub        │  │  - Proximity Alert Evaluator│
      │  - River Job Queue Tables   │  │  - Refresh Token Allow-List │  │  - Scheduled Reminders      │
      └─────────────────────────────┘  └─────────────────────────────┘  └─────────────────────────────┘
```

### 2.2 Komponen

| Komponen | Teknologi | Tanggung Jawab |
| --- | --- | --- |
| Frontend | Next.js + TypeScript, Tailwind CSS + shadcn/ui, TanStack Query, React Hook Form + Zod | Seluruh antarmuka web: booking tamu, riwayat customer, dashboard owner, konsol operator, dan layar display. |
| API | Go + Hertz | REST `/api/v1`: validasi, autentikasi, otorisasi, penentuan tenant, use case, dan akses database. |
| WebSocket gateway | Go (`github.com/coder/websocket`), berjalan di proses API | Menyiarkan event antrean ke operator, layar display, dan halaman status pelanggan. |
| Worker | Go, River, outbox relay | Mengirim email, menjalankan reminder dan tugas terjadwal, menerbitkan event outbox ke Redis. |
| PostgreSQL | Versi stable terbaru yang didukung, di-pin | Data utama, constraint GiST, dan RLS. Antrean job River juga tersimpan di sini. |
| Redis | Versi stable terbaru yang didukung, di-pin | Cache data referensi, rate limiting, pub/sub event real-time, refresh token dan token sekali pakai. |
| Email | SMTP (misal Mailpit di dev, SMTP provider di prod) | Pengiriman email booking, perubahan jadwal, dan pengingat berjenjang. |
| Observability | Prometheus, Grafana, OpenTelemetry | Metrics, dashboard, dan distributed tracing. |
| Infrastruktur | Docker, Docker Compose, GitHub Actions, VPS | Packaging, CI/CD, dan automated deployment. |

### 2.3 Alur Runtime Utama

**A. Booking oleh Tamu**
1. Browser memanggil `GET /api/v1/appointments/availability` untuk melihat slot.
2. Browser memanggil `POST /api/v1/appointments` dengan header `Idempotency-Key`.
3. API membuka satu transaksi dan mengisi konteks tenant. Di dalamnya: customer dicari atau dibuat berdasarkan email, staff ditetapkan bila belum dipilih, appointment disimpan, notifikasi konfirmasi dan tiga reminder dibuat beserta job-nya di River, dan audit log dicatat. Lalu commit.
4. Jika slot sudah diambil request lain, exclusion constraint menolak insert dan API membalas `409 SLOT_UNAVAILABLE`.
5. Worker mengirim email konfirmasi secara asynchronous.

**B. Check-in**
1. Pelanggan membuka halaman check-in (dari website, hasil scan QR, atau memasukkan kode booking) dan mengirim `POST /api/v1/check-in`.
2. API memeriksa check-in window cabang, lalu dalam satu transaksi: mengunci baris appointment, memastikan belum ada tiket, menaikkan counter harian, membuat queue ticket dengan prioritas dari `branch_settings`, mengubah status appointment menjadi `CHECKED_IN`, dan menulis event outbox `queue.ticket_created`.
3. Respons berisi tiket dan `tracking_token` untuk memantau antrean.
4. Jika pelanggan terlambat (>15 menit), staf loket dapat menggunakan `POST /api/v1/check-in/override` untuk menerbitkan tiket walk-in.

**C. Call Next**
1. Operator mengirim `POST /api/v1/queue/call-next` dengan loket yang sedang dipakai.
2. Dalam satu transaksi API memilih tiket `WAITING` dengan prioritas tertinggi memakai `FOR UPDATE SKIP LOCKED`, mengubahnya menjadi `CALLED`, mengisi loket dan staff, lalu menulis event outbox `queue.ticket_called`.
3. Outbox relay di worker menerbitkan event ke Redis. Setiap instance API yang berlangganan meneruskannya lewat WebSocket ke layar display, konsol operator, dan halaman status pelanggan.
4. Worker mengevaluasi ambang `notify_before_queue_count` dan membuat notifikasi in-app untuk pelanggan yang mendekati giliran.

---

## 3. Backend

### 3.1 Struktur Proyek

```
quevia-backend/
├── cmd/
│   ├── api/main.go
│   └── worker/main.go
├── internal/
│   ├── auth/
│   ├── organization/
│   ├── branch/
│   ├── customer/
│   ├── service/
│   ├── staff/
│   ├── appointment/
│   ├── queue/
│   ├── notification/
│   ├── analytics/
│   ├── subscription/
│   ├── realtime/
│   └── platform/
│       ├── database/      # pool pgx, WithTx, konteks tenant RLS
│       ├── cache/
│       ├── outbox/
│       ├── idempotency/
│       ├── audit/
│       ├── logger/
│       └── middleware/
├── api/openapi.yaml
├── migrations/
├── sql/                   # query sqlc per modul
├── configs/
├── Dockerfile
└── go.mod
```

### 3.2 Aturan Layering

- Setiap modul memisahkan **handler** (parsing, validasi, otorisasi, pemetaan DTO), **service/use case** (aturan bisnis dan batas transaksi), **repository** (query sqlc), dan **model**.
- Business logic dilarang berada di handler HTTP.
- Modul tidak membaca tabel milik modul lain secara langsung. Kebutuhan lintas modul dipenuhi lewat interface service modul pemilik atau lewat event asynchronous.
- Transaksi dibuka oleh use case melalui helper `WithTx`, yang juga mengisi `app.current_tenant_id` dengan `set_config(..., true)` agar RLS PostgreSQL aktif.

---

## 4. Real-Time (WebSocket)

### 4.1 Koneksi dan Autentikasi

Satu endpoint melayani semua klien: `/ws/v1/branches/:branchGuid/queue`. Token tidak dikirim lewat URL. Setelah terhubung, klien harus mengirim pesan `auth` dalam 5 detik; jika tidak, koneksi diputus.

| Klien | Kredensial | Topik yang Diizinkan |
| --- | --- | --- |
| Layar display | Token display (hasil tukar token layar) | `board` cabang tersebut |
| Operator dan admin | Access token dengan hak akses ke cabang | `board` cabang tersebut |
| Pelanggan atau tamu | `tracking_token` dari respons check-in / walk-in | `ticket` miliknya sendiri |

Jika koneksi terputus, klien display dan pelanggan menampilkan banner indikator visual "Menghubungkan kembali..." dan menyinkronkan ulang snapshot REST setelah tersambung kembali.

---

## 5. Frontend

### 5.1 Struktur Rute

Satu aplikasi Next.js (App Router) dipisah menjadi route group per jenis pengguna:

```
quevia-frontend/
├── app/
│   ├── (public)/               # landing, tenant katalog, booking tamu, check-in, tracking tiket
│   │   ├── t/[tenantSlug]/
│   │   ├── booking/
│   │   ├── check-in/
│   │   └── track/[ticketGuid]/
│   ├── (customer)/             # riwayat booking & tiket customer terdaftar
│   ├── (dashboard)/            # owner dan admin cabang
│   │   ├── dashboard/ branches/ services/ staff/
│   │   └── appointments/ queue/ analytics/ settings/
│   ├── (operator)/counter/     # konsol operator loket
│   └── display/[branchGuid]/   # layar publik antrean kiosk
├── components/                 # ui, forms, calendar, queue, dashboard
├── features/                   # auth, appointment, queue, branch, service, staff
├── hooks/  lib/  services/  schemas/  types/
```

---

## 6. Autentikasi dan Otorisasi

### 6.1 Jenis Token
- **Access token**: JWT bertanda tangan (15 menit) berisi user guid, tenant, role, dan branch scope; disimpan di memori klien.
- **Refresh token**: String acak dirotasi (14 hari); disimpan di cookie httpOnly, terdaftar pada allow-list Redis `rt:<jti>`.
- **Guest token**: JWT terbatas pada satu appointment (30 menit); disimpan di memori klien.
- **Ticket token**: JWT terbatas pada satu tiket antrean (`tracking_token`); berlaku sampai akhir hari antrean.
- **Display token**: Hash tersimpan di `display_screens.access_token_hash`, ditukar menjadi display session JWT read-only board.

Role database `quevia_admin` digunakan untuk proses migrasi dan bypass RLS khusus platform Super Admin.

---

## 7. Kontrak API

### 7.1 Konvensi
- Base path: `/api/v1`, format JSON UTF-8.
- Autentikasi: Header `Authorization: Bearer <token>`.
- Identitas publik di URL dan body menggunakan `guid` (UUID v4); `id` internal tidak pernah diekspos.
- Header `Idempotency-Key` wajib pada `POST /appointments`, `POST /queue/walk-in`, dan `POST /check-in`.

### 7.2 Daftar Endpoint Terpilih

**Autentikasi & Organisasi**
- `POST /api/v1/auth/login` (Public)
- `POST /api/v1/auth/refresh` (Cookie)
- `POST /api/v1/tenants` (Public) — Pendaftaran tenant dan cabang pertama
- `GET, PATCH /api/v1/tenant` (Admin)

**Cabang & Layanan**
- `GET, POST /api/v1/branches` (Staff, Admin)
- `GET, PUT /api/v1/branches/:guid/settings` (Staff, Admin)
- `GET, POST /api/v1/services` (Staff, Admin)
- `GET, POST /api/v1/staff` (Staff, Admin)
- `PUT /api/v1/staff/:guid/services` (Admin)
- `GET, PUT /api/v1/staff/:guid/schedules` (Staff, Admin)

**Appointment & Booking**
- `GET /api/v1/appointments/availability` (Public) — Cek slot tersedia
- `POST /api/v1/appointments` (Public) — Booking mandiri (Idempotent)
- `POST /api/v1/public/appointments/lookup` (Public) — Lookup booking via kode & email
- `PATCH /api/v1/appointments/:guid` (Guest, Auth, Staff) — Reschedule
- `POST /api/v1/appointments/:guid/cancel` (Guest, Auth, Staff) — Cancel booking
- `POST /api/v1/appointments/:guid/reassign` (Admin) — **Reassign staff pada janji temu aktif** (CR-04)

**Check-in & Antrean**
- `POST /api/v1/check-in` (Public, Staff) — Check-in QR/Web/Kode (Idempotent)
- `POST /api/v1/check-in/override` (Staff) — **Manual check-in override untuk appointment terlambat** (CR-06)
- `POST /api/v1/queue/walk-in` (Public, Staff) — Ambil tiket walk-in (Idempotent)
- `GET /api/v1/queue/board` (Staff, Display) — Snapshot board antrean
- `GET /api/v1/queue/tickets/:guid/status` (Ticket) — Status pelacakan antrean pelanggan
- `POST /api/v1/queue/call-next` (Staff) — Panggil antrean berikutnya
- `POST /api/v1/queue/tickets/:guid/recall` (Staff) — **Panggil ulang tiket (termasuk tiket SKIPPED)** (CR-02)
- `POST /api/v1/queue/tickets/:guid/skip` (Staff) — Lewati tiket
- `POST /api/v1/queue/tickets/:guid/start` (Staff) — Mulai layanan
- `POST /api/v1/queue/tickets/:guid/complete` (Staff) — Selesaikan layanan
- `POST /api/v1/queue/tickets/:guid/no-show` (Staff) — Tandai NO_SHOW

**Contoh Payload Booking Response**:
```json
{
  "data": {
    "guid": "550e8400-e29b-41d4-a716-446655440000",
    "booking_code": "QV-8K2M4P",
    "qr_token": "a1b2c3d4e5f67890abcdef1234567890",
    "status": "CONFIRMED",
    "start_at": "2026-11-12T09:00:00+07:00",
    "end_at": "2026-11-12T09:30:00+07:00",
    "staff": {
      "guid": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "display_name": "Budi Santoso"
    }
  }
}
```

---

## 8. Dokumen Terkait

1. **PRD Quevia v1.2** (`PRD — Quevia Smart Queue & Appointment Management.md`)
2. **ERD Quevia Database Design** (`ERD — Quevia Database Design.md`)

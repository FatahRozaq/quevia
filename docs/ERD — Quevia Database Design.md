# ERD — Quevia: Database Design

Versi 1.1 · 10 November 2026 · Penulis: Sarah & Fatah · Mengacu pada PRD Quevia v1.2

## 1. Ringkasan

Skema database Quevia MVP terdiri dari 26 tabel PostgreSQL dalam 7 domain operasional: 24 tabel inti untuk kebutuhan MVP dan 2 tabel pendukung untuk simulasi langganan SaaS. Seluruh tabel bisnis menerapkan 8 kolom standar audit (id, guid, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by), soft delete, serta isolasi data multi-tenant berlapis: penyaringan di level aplikasi dan penegakan mutlak melalui PostgreSQL Row-Level Security (RLS).

Tiga jaminan kritis yang dipegang langsung oleh lapisan database:

1. **Anti Double Booking Mutlak**: Tidak ada dua appointment aktif milik staff yang sama pada waktu yang tumpang tindih (`ex_appointments_no_overlap` berbasis GiST extension `btree_gist`).
2. **Nomor Antrean Unik & Berurutan**: Tidak ada nomor antrean ganda per cabang, layanan, dan tanggal melalui partial unique index dan row locking transaksi counter (`queue_counters`).
3. **Isolasi Multi-Tenant Tak Terbobol**: Data satu tenant tidak dapat terbaca atau termodifikasi oleh tenant lain (`FORCE ROW LEVEL SECURITY`).

---

## 2. Konvensi Database

### 2.1 Kolom Standar Audit

Setiap tabel bisnis memiliki 8 kolom standar berikut:

| Kolom | Tipe | Keterangan |
| --- | --- | --- |
| `id` | BIGINT GENERATED ALWAYS AS IDENTITY | Primary Key internal untuk relasi foreign key dan join berkecepatan tinggi. |
| `guid` | UUID NOT NULL UNIQUE DEFAULT gen_random_uuid() | Identitas publik eksternal yang diekspos ke REST API dan URL publik. |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT NOW() | Waktu pembuatan data dalam UTC. |
| `created_by` | BIGINT NULL FK users(id) | Pembuat data; aksi tamu dicatat memakai reserved user `guest`. |
| `updated_at` | TIMESTAMPTZ NOT NULL DEFAULT NOW() | Waktu pembaruan data terakhir. |
| `updated_by` | BIGINT NULL FK users(id) | Pengubah data terakhir. |
| `deleted_at` | TIMESTAMPTZ NULL | Penanda soft delete; NULL berarti baris berstatus aktif. |
| `deleted_by` | BIGINT NULL FK users(id) | Pengguna yang melakukan soft delete. |

*Pengecualian*: Tabel append-only `audit_logs` hanya memakai 4 kolom audit pertama. Tabel operasional `outbox_events` dan `idempotency_keys` tidak memakai `deleted_at`/`deleted_by` karena dibersihkan permanen oleh worker retention job.

---

## 3. Daftar Tabel

| Domain | Nama Tabel | Deskripsi Fungsi | RLS Policy |
| --- | --- | --- | --- |
| Tenant & Identity | `tenants` | Organisasi bisnis yang berlangganan Quevia | Ya (id) |
| Tenant & Identity | `branches` | Cabang bisnis dan konfigurasi zona waktu IANA | Ya |
| Tenant & Identity | `users` | Akun pengguna global (Admin, Staff, Customer ber-akun, reserved guest) | Tidak (Global) |
| Tenant & Identity | `memberships` | Asosiasi user, tenant, role (TENANT_ADMIN, STAFF), dan cabang | Ya |
| Tenant & Identity | `invitations` | Undangan pengguna ke dalam organisasi tenant | Ya |
| Master & Konfigurasi | `service_categories`| Pengelompokan kategori layanan bisnis | Ya |
| Master & Konfigurasi | `services` | Katalog layanan per cabang, durasi pengerjaan, dan buffer time | Ya |
| Master & Konfigurasi | `staff_profiles` | Profil petugas per cabang dan penugasan spesialisasi | Ya |
| Master & Konfigurasi | `staff_services` | Tabel relasi N:N antara staff dan layanan yang dapat dikerjakan | Ya |
| Master & Konfigurasi | `service_counters` | Loket fisik atau meja pelayanan pelanggan | Ya |
| Master & Konfigurasi | `display_screens` | Perangkat layar antrean kiosk publik per cabang | Ya |
| Master & Konfigurasi | `branch_settings` | Konfigurasi aturan antrean, toleransi keterlambatan, dan ambang notifikasi | Ya |
| Penjadwalan | `business_hours` | Jam buka dan tutup operasional mingguan cabang | Ya |
| Penjadwalan | `staff_schedules` | Jam kerja mingguan dan jam istirahat masing-masing staff | Ya |
| Penjadwalan | `schedule_exceptions`| Pengecualian jadwal: hari libur cabang, cuti staff, jam khusus | Ya |
| Customer & Appointment| `customers` | Data profil pelanggan (terdaftar maupun tamu) | Ya |
| Customer & Appointment| `appointments` | Janji temu terjadwal, kode booking `QV-`, dan token QR | Ya |
| Antrean (Queue Engine)| `queue_counters` | Generator counter nomor antrean harian per cabang dan layanan | Ya |
| Antrean (Queue Engine)| `queue_tickets` | Tiket antrean harian (walk-in maupun appointment ter-check-in) | Ya |
| Notifikasi & Sistem | `notification_templates`| Template pesan per event (Email & In-App) | Ya |
| Notifikasi & Sistem | `notifications` | Riwayat dan antrean pengiriman notifikasi berjadwal | Ya |
| Notifikasi & Sistem | `audit_logs` | Riwayat aktivitas dan jejak audit kepatuhan (append-only) | Ya |
| Notifikasi & Sistem | `outbox_events` | Tabel outbox untuk diseminasi event real-time via Redis Pub/Sub | Ya |
| Notifikasi & Sistem | `idempotency_keys` | Penyimpanan payload request untuk menjamin idempotensi operasi | Ya |
| Langganan (Simulasi)| `subscription_plans`| Paket langganan platform (Free, Pro, Business) | Tidak (Global) |
| Langganan (Simulasi)| `tenant_subscriptions`| Status paket aktif langganan per tenant | Ya |

---

## 4. Definisi Tabel Kunci (DDL & Constraint)

### 4.1 Tabel `branch_settings`
Tabel ini memuat konfigurasi operasional antrean dan janji temu per cabang:

```sql
CREATE TABLE branch_settings (
    id                          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                        UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT NOT NULL REFERENCES tenants(id),
    branch_id                   BIGINT NOT NULL REFERENCES branches(id),

    change_cutoff_minutes       INTEGER NOT NULL DEFAULT 180,  -- batas cancel/reschedule (3 jam)
    check_in_open_before_minutes INTEGER NOT NULL DEFAULT 30,  -- jendela check-in dibuka
    check_in_close_after_minutes INTEGER NOT NULL DEFAULT 15,  -- jendela check-in ditutup
    no_show_after_minutes       INTEGER NOT NULL DEFAULT 10,   -- toleransi panggil sebelum no-show
    slot_step_minutes           SMALLINT NOT NULL DEFAULT 30,  -- interval pencarian slot
    appointment_priority        SMALLINT NOT NULL DEFAULT 2,   -- bobot prioritas appointment
    walk_in_priority            SMALLINT NOT NULL DEFAULT 1,   -- bobot prioritas walk-in
    late_grace_minutes          INTEGER NULL,                  -- toleransi telat sebelum turun prioritas
    notify_before_queue_count   SMALLINT NOT NULL DEFAULT 2,   -- ambang pemicu QUEUE_ALMOST_CALLED (CR-05)

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  BIGINT REFERENCES users(id),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                  BIGINT REFERENCES users(id),
    deleted_at                  TIMESTAMPTZ,
    deleted_by                  BIGINT REFERENCES users(id),

    CONSTRAINT uq_branch_settings_branch UNIQUE (branch_id) WHERE deleted_at IS NULL
);
```

### 4.2 Tabel `appointments`
Tabel ini menampung seluruh booking pelanggan dengan proteksi exclusion constraint:

```sql
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE appointments (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid          UUID NOT NULL DEFAULT gen_random_uuid(),

    tenant_id     BIGINT NOT NULL REFERENCES tenants(id),
    branch_id     BIGINT NOT NULL REFERENCES branches(id),
    customer_id   BIGINT NOT NULL REFERENCES customers(id),
    service_id    BIGINT NOT NULL REFERENCES services(id),
    staff_id      BIGINT NOT NULL REFERENCES staff_profiles(id),

    booking_code  VARCHAR(30) NOT NULL,            -- format QV-XXXXXX
    qr_token      VARCHAR(64) NOT NULL,            -- token unik untuk check-in QR
    start_at      TIMESTAMPTZ NOT NULL,
    end_at        TIMESTAMPTZ NOT NULL,
    status        VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
    notes         TEXT,
    rescheduled_count SMALLINT NOT NULL DEFAULT 0,
    cancelled_at  TIMESTAMPTZ,
    cancel_reason VARCHAR(255),

    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    BIGINT REFERENCES users(id),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by    BIGINT REFERENCES users(id),
    deleted_at    TIMESTAMPTZ,
    deleted_by    BIGINT REFERENCES users(id),

    CONSTRAINT uq_appointments_guid UNIQUE (guid),
    CONSTRAINT ck_appointments_time CHECK (end_at > start_at),
    CONSTRAINT ck_appointments_status CHECK (
        status IN ('CONFIRMED','CHECKED_IN','COMPLETED','CANCELLED','NO_SHOW')
    ),
    CONSTRAINT ex_appointments_no_overlap EXCLUDE USING gist (
        staff_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (status IN ('CONFIRMED','CHECKED_IN') AND deleted_at IS NULL)
);

CREATE UNIQUE INDEX uq_appointments_booking_code
    ON appointments (booking_code) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_appointments_qr_token
    ON appointments (qr_token) WHERE deleted_at IS NULL;
```

### 4.3 Tabel `queue_tickets`
Tabel ini menampung tiket antrean harian:

```sql
CREATE TABLE queue_tickets (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid           UUID NOT NULL DEFAULT gen_random_uuid(),

    tenant_id      BIGINT NOT NULL REFERENCES tenants(id),
    branch_id      BIGINT NOT NULL REFERENCES branches(id),
    customer_id    BIGINT REFERENCES customers(id),    -- NULL = walk-in anonim
    service_id     BIGINT NOT NULL REFERENCES services(id),
    appointment_id BIGINT REFERENCES appointments(id),  -- NULL = walk-in
    counter_id     BIGINT REFERENCES service_counters(id),
    staff_id       BIGINT REFERENCES staff_profiles(id),

    queue_date     DATE NOT NULL,
    queue_number   INTEGER NOT NULL,
    queue_code     VARCHAR(20) NOT NULL,               -- contoh: A-023
    source         VARCHAR(20) NOT NULL,               -- WALK_IN | APPOINTMENT
    priority       INTEGER NOT NULL DEFAULT 0,
    status         VARCHAR(30) NOT NULL DEFAULT 'WAITING',

    check_in_method VARCHAR(20),                       -- WEB | QR | BOOKING_CODE | OVERRIDE
    checked_in_at  TIMESTAMPTZ,
    called_at      TIMESTAMPTZ,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    recall_count   SMALLINT NOT NULL DEFAULT 0,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     BIGINT REFERENCES users(id),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by     BIGINT REFERENCES users(id),
    deleted_at     TIMESTAMPTZ,
    deleted_by     BIGINT REFERENCES users(id),

    CONSTRAINT uq_queue_tickets_guid UNIQUE (guid),
    CONSTRAINT ck_queue_tickets_status CHECK (
        status IN ('WAITING','CALLED','SERVING','COMPLETED','SKIPPED','CANCELLED','NO_SHOW')
    ),
    CONSTRAINT ck_queue_tickets_source_appointment CHECK (
        (source = 'APPOINTMENT') = (appointment_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_queue_tickets_appointment
    ON queue_tickets (appointment_id)
    WHERE appointment_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX uq_queue_tickets_number
    ON queue_tickets (branch_id, service_id, queue_date, queue_number)
    WHERE deleted_at IS NULL;
```

---

## 5. Row-Level Security (RLS) dan Akses Database

Setiap koneksi transaksi aplikasi mengisi konteks tenant di PostgreSQL:

```sql
SELECT set_config('app.current_tenant_id', $tenant_id, true);
```

Setiap tabel yang menyimpan data spesifik tenant diaktifkan RLS dengan kebijakan isolasi mutlak:

```sql
ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON appointments
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
```

Role database untuk aplikasi dijalankan tanpa hak `BYPASSRLS`. Role administratif `quevia_admin` digunakan khusus untuk pipeline migrasi skema dan dashboard monitoring platform Super Admin.

---

## 6. Dokumen Terkait

1. **PRD Quevia v1.2** (`PRD — Quevia Smart Queue & Appointment Management.md`)
2. **System Architecture & API Contract — Quevia** (`System Architecture & API Contract — Quevia.md`)
3. **PRD Change Log** (`docs/04-prd-change-log.md`)

# PRD Assessment Report: Quevia (formerly QFlow)

**Tanggal**: 10 November 2026  
**Auditor / Evaluator**: Sarah (Technical Product Owner & Requirements Specialist) / Senior PM & System Analyst  
**Subjek**: Evaluasi Awal PRD, System Architecture, dan ERD Platform Smart Queue & Appointment Management  
**Status**: Phase 1 Deliverable — Completed  

---

## 1. Executive Summary & Project Context

### 1.1 Konteks Produk
Produk yang diaudit adalah platform SaaS multi-tenant berbasis web responsif yang dirancang untuk mengintegrasikan dua domain operasional kritis dalam bisnis berbasis layanan:
1. **Appointment Management**: Penjadwalan mandiri oleh pelanggan (baik terdaftar maupun tamu/guest) dengan algoritma anti-double booking.
2. **Queue Management**: Alur antrean harian di lokasi fisik (check-in, tiket antrean, pemanggilan loket, tampilan display publik real-time, dan notifikasi omnichannel email & in-app).

Domain acuan demo yang dipilih adalah bengkel otomotif/service center, namun model data dan arsitektur dirancang secara generik untuk dapat diadopsi oleh klinik, salon/barbershop, dan kantor pelayanan publik.

### 1.2 Dokumen yang Diaudit
1. `PRD — QFlow Smart Queue & Appointment Management.md` (v1.1 Draft)
2. `System Architecture & API Contract — QFlow.md` (v1.0 Draft)
3. `ERD — QFlow Database Design.md` (v1.0 Draft)

### 1.3 Permintaan Khusus Rebranding
Pengguna telah menginstruksikan perubahan identitas merek (*rebranding*) dari **QFlow** menjadi **Quevia** di seluruh ekosistem dokumentasi (3 file Markdown) dan artefak teknis terkait.

---

## 2. Analisis Tujuan, Ruang Lingkup, dan Batasan Sistem

### 2.1 Analisis Tujuan Produk (Goals)
Dokumen menetapkan 7 tujuan strategis (G-01 s.d. G-07):
- **G-01 (Mandiri & Anti Double Booking)**: Memungkinkan booking mandiri dengan pencegahan konkurensi di database.
- **G-02 (Unifikasi Antrean)**: Menggabungkan antrean terjadwal (appointment) dan walk-in dalam satu mesin antrean berbasis aturan prioritas dinamis.
- **G-03 (Sinkronisasi Real-Time)**: Penyebaran status antrean < 1 detik ke dashboard operator, kiosk display, dan mobile tracking.
- **G-04 (Notifikasi Otomatis)**: Pengiriman konfirmasi, reminder berjenjang (T-24h, T-60m, T-15m), dan panggilan antrean.
- **G-05 (Visibilitas Bisnis)**: Dashboard analitik performa cabang, staff, dan beban waktu tunggu.
- **G-06 (Isolasi Multi-Tenant Mutlak)**: Tidak ada kebocoran data antar-tenant baik di tingkat aplikasi maupun database (PostgreSQL RLS).
- **G-07 (Portofolio Rekayasa Perangkat Lunak)**: Menjadi bukti kompetensi arsitektur Fullstack, konkurensi, dan sistem terdistribusi ringan.

### 2.2 Batasan Ruang Lingkup (Scope Boundaries)
- **In-Scope (MVP)**:
  - 8 Modul Inti: AUTH (Autentikasi & Org), SVC (Layanan & Staff), APT (Appointment), QUE (Antrean), DSP (Layar Display), NTF (Notifikasi Email & In-App), ANL (Analitik Dasar), SET (Pengaturan Bisnis).
  - Web responsif lintas peran (Customer/Guest, Operator/Staff, Tenant Admin, Super Admin, Kiosk Screen).
  - PostgreSQL dengan constraint `btree_gist` dan RLS, Redis Cache/PubSub, Worker River, Hertz API Go, Next.js App Router.
- **Out-of-Scope (Non-Goals Terverifikasi)**:
  - Integrasi payment gateway nyata (hanya simulasi plan langganan).
  - Algoritma prediksi beban berbasis machine learning / AI.
  - Arsitektur microservices (arsitektur dipertahankan sebagai Modular Monolith).
  - Aplikasi native iOS/Android atau Flutter (fokus website responsif).
  - Integrasi WhatsApp gateway (email sebagai kanal eksternal primer MVP).

---

## 3. Analisis Pengguna, Hak Akses, dan Hierarki Organisasi

Sistem memetakan model relasi hierarkis tenant yang solid:
```
Quevia Platform (Global Scope: Super Admin)
└── Tenant / Organisasi (Business Owner / Tenant Admin)
    ├── Branch / Cabang (Timezone, Business Hours, Counters, Displays)
    │   ├── Staff Profiles & Schedules (Scoped to Branch)
    │   ├── Services & Categories
    │   └── Daily Queue Engine (Counters & Tickets)
    └── Customers (Global/Tenant shared, Guest supported)
```

| Peran | Ruang Lingkup | Mekanisme Autentikasi | Catatan Kritis |
| --- | --- | --- | --- |
| **Super Admin** | Platform-wide | JWT `platform_role = SUPER_ADMIN` | Bypass RLS via role DB khusus (`qflow_admin`/`quevia_admin`). |
| **Tenant Admin** | Tenant-wide / Multi-branch | JWT + Membership scoped | Memiliki kendali konfigurasi penuh pada tenant. |
| **Staff / Operator** | Single Branch | JWT + Membership branch-bound | Dibatasi hanya dapat mengelola antrean dan tiket di cabang tugasnya. |
| **Customer (Akun)** | Personal Profile | JWT Customer | Memantau riwayat booking dan menerima notifikasi in-app tersimpan. |
| **Guest (Tamu)** | Single Booking / Ticket | Guest Token (Lookup via booking code + email) / Ticket Tracking Token | Tidak memerlukan registrasi akun; email wajib untuk verifikasi dan reminder. |
| **Queue Display** | Single Branch Board | Display Session Token (ditukar dari hashed hardware access token) | Read-only board, auto-reconnect, kiosk UI tanpa interaksi login berulang. |

---

## 4. Quality Assessment Scorecard (100-Point Scale)

Evaluasi sistematis terhadap PRD eksisting mengacu pada framework 5 dimensi kualitas:

```
📊 Requirements Quality Score: 84/100

Breakdown:
- Business Value & Goals:     23/30
- Functional Requirements:    21/25
- User Experience:            16/20
- Technical Constraints:      14/15
- Scope & Priorities:         10/10

Status: Good / Solid Foundation. Butuh penyempurnaan terarah untuk mencapai threshold plannable (≥ 90).
```

### Rincian Evaluasi per Dimensi

#### 4.1 Business Value & Goals (23/30)
- **Problem Statement (8/10)**: Masalah operasional konvensional dijabarkan secara jelas (ketidakpastian waktu, double booking, no-show). *Gap*: Belum menyertakan data baseline terukur (misal estimasi % tingkat no-show industri bengkel atau rata-rata menit waktu tunggu terbuang) serta dampak finansial langsung.
- **Success Metrics & KPIs (8/10)**: Metrik teknis sangat presisi (0 double booking, 0 duplicate ticket, 0 tenant leak, p95 < 1s sync, p95 < 300ms availability). *Gap*: Metrik nilai bisnis (misal target penurunan no-show dari X% menjadi Y%, customer retention rate) belum memiliki guardrail/counter-metric (misal: kecepatan pelayanan staff tidak boleh mengorbankan kualitas layanan).
- **Expected Outcomes & ROI (7/10)**: Value proposition efisiensi operasional jelas, namun justifikasi ROI komersial untuk pemilik bisnis masih minimal karena modul billing berstatus simulasi/opsional.

#### 4.2 Functional Requirements (21/25)
- **User Stories & Acceptance Criteria (8/10)**: User stories (US-01 s.d. US-09) mencakup alur primer secara baik. *Gap*: Acceptance criteria belum merinci skenario kegagalan sekunder secara formal (misal: jika staff tiba-tiba dinonaktifkan saat memiliki janji aktif, atau perlakuan saat pelanggan datang tepat 1 detik setelah jendela check-in ditutup).
- **Feature Descriptions & Workflows (9/10)**: Penjelasan alur booking, check-in QR/kode, call next, dan sinkronisasi display tersambung sangat rapi dengan dokumen arsitektur dan ERD.
- **Edge Cases & Error Handling (4/5)**: Penanganan konkurensi (row lock, idempotency key, exclusion constraint) luar biasa lengkap. *Gap*: Penanganan operasional seperti pengalihan antrean antar-loket (*counter transfer*) atau penanganan gangguan hardware display belum terdokumentasi di PRD.

#### 4.3 User Experience (16/20)
- **User Personas (6/8)**: Hak akses dan peran didefinisikan secara struktural. *Gap*: Belum ada profil persona perilaku/empati (misal: karakteristik pemilik bengkel yang tidak terbiasa dengan UI rumit, atau pelanggan terburu-buru) beserta *Jobs-to-be-Done* (JTBD).
- **User Journey & Interaction Flows (6/7)**: Alur fungsional dipetakan dari buka website hingga selesai dilayani. *Gap*: Belum merinci penanganan *empty states*, indikator status offline/reconnecting pada UI pelanggan, dan instruksi visual pada layar display kiosk.
- **UI/UX Preferences & Constraints (4/5)**: Pendekatan responsif dan kebijakan audio autoplay browser sudah diantisipasi secara tepat. *Gap*: Belum menyertakan standar kepatuhan aksesibilitas formal (seperti WCAG 2.2 AA).

#### 4.4 Technical Constraints (14/15)
- **Performance Requirements (5/5)**: Target p95 dan pengujian konkurensi (100 concurrent requests) terdefinisi secara kuantitatif.
- **Security & Compliance (5/5)**: Pengamanan isolasi multi-tenant berlapis (Middleware + RLS + Argon2id + Token Scoping + Uniform Error Handling) sangat matang.
- **Integration Requirements (4/5)**: Pola outbox, River background job, dan WebSocket terdefinisi baik. *Gap*: Spesifikasi fallback jika penyedia SMTP mengalami throttling atau down berkepanjangan belum memiliki sirkuit pemutus (*circuit breaker*).

#### 4.5 Scope & Priorities (10/10)
- **MVP Definition (5/5)**: Batasan antara kebutuhan inti dan non-goals sangat tegas dan defensibel.
- **Phased Delivery (3/3)**: Roadmap 4 fase (Core Foundation, Core Business, Real-Time & Automation, Production Readiness) terstruktur logis.
- **Priority Rankings (2/2)**: Klasifikasi MoSCoW (M, S, C) diterapkan secara konsisten pada seluruh requirement.

---

## 5. Identifikasi Asumsi dan Pertanyaan Terbuka

### 5.1 Asumsi yang Perlu Divalidasi
1. **Asumsi Domain Demo**: Bengkel otomotif dipilih sebagai demo implementasi pertama. Model data generik diasumsikan cukup fleksibel tanpa perlu kolom khusus bengkel (seperti nomor polisi kendaraan / VIN) di tabel inti; jika dibutuhkan, ditampung via metadata/notes atau JSONB.
2. **Asumsi Kanal Komunikasi**: Email diasumsikan cukup untuk skenario MVP, dengan asumsi pelanggan memeriksa email di smartphone saat menerima kode booking atau pengingat jadwal.
3. **Asumsi Perilaku Tamu**: Pelanggan walk-in dianggap bersedia memantau layar display fisik atau memindai QR tiket di ponsel mereka untuk melihat estimasi antrean tanpa harus membuat akun.

### 5.2 Pertanyaan Terbuka (Unresolved Technical & Business Questions)
1. **Ambang Batas "Queue Almost Called"**: Event `QUEUE_ALMOST_CALLED` sudah didefinisikan di arsitektur, namun tabel `branch_settings` di ERD belum memiliki kolom konfigurasi jumlah antrean pemicu (misal: panggil notifikasi saat sisa 2 antrean di depan).
2. **Ketahanan Refresh Token Sesi**: Refresh token saat ini disimpan di Redis (`rt:<jti>`). Jika Redis di-restart tanpa persistensi disk, seluruh pengguna yang sedang login akan ter-logout paksa. Apakah diperlukan fallback tabel `refresh_tokens` di PostgreSQL?
3. **Pemberitahuan Keterlambatan Layanan**: Jika suatu pengerjaan di loket memakan waktu jauh lebih lama dari estimasi durasi layanan, bagaimana sistem memperbarui estimasi waktu tunggu (*estimated wait time*) bagi antrean di belakangnya?

---

## 6. Rencana Aksi Rebranding (QFlow → Quevia)

Untuk memenuhi instruksi pengguna terkait perubahan brand pada 3 file Markdown:
- **File PRD**: Mengubah judul, referensi produk, slug contoh, deskripsi platform, dan kode booking default (`QF-` menjadi `QV-`).
- **File Architecture**: Mengubah nama repo/folder (`qflow-backend` → `quevia-backend`, `qflow-frontend` → `quevia-frontend`), database user (`qflow_admin` → `quevia_admin`), dan header referensi.
- **File ERD**: Mengubah judul, referensi relasi, role admin, dan artefak terkait.
- *Aturan Eksekusi*: Sesuai Change Management, perubahan teks pada 3 file akan diformulasikan dalam Change Proposal (Phase 3/4) dan dieksekusi secara in-place pada Phase 4 (PRD Revision) setelah audit kebutuhan selesai.

---

## 7. Kesimpulan & Rekomendasi Transisi ke Phase 2

PRD saat ini telah memiliki arsitektur fungsional dan teknis yang sangat solid (Skor 84/100). Untuk melangkah ke skor 90+ dan siap implementasi (*engineering-ready*), audit kebutuhan pada **Phase 2 (Requirement Audit)** akan difokuskan pada:
1. Menemukan missing edge-cases pada operasional loket dan pembatalan mendadak.
2. Memperjelas acceptance criteria pada skenario batas (*boundary testing*).
3. Melengkapi NFR dengan target numerik standar industri (WCAG 2.2 AA, recovery time objective).
4. Menyusun Change Log dan Proposal Rebranding terstruktur untuk migrasi dari QFlow ke Quevia.

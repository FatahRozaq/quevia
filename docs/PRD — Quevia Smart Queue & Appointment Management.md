# PRD — Quevia: Smart Queue & Appointment Management

Versi 1.2 · 10 November 2026 · Penulis: Sarah (Product Owner) & Fatah · Status: Approved

## Riwayat Perubahan Dokumen (Doc Change History)

| Versi | Tanggal | Penulis | Deskripsi Perubahan |
| --- | --- | --- | --- |
| 1.0 | 01 November 2026 | Fatah | Draft awal spesifikasi produk (QFlow). |
| 1.1 | 10 November 2026 | Fatah | Konsolidasi aturan bisnis, kolom audit, dan konvensi RLS. |
| **1.2** | **10 November 2026** | **Sarah & Fatah** | **Rebranding resmi ke Quevia**; penutupan celah siklus tiket `SKIPPED`; sinkronisasi dua arah status pembatalan tiket terhadap appointment; proteksi penonaktifan staff berjadwal aktif (FR-SVC-06, BR-19); diskresi *manual check-in override* untuk pelanggan terlambat (BR-20); penambahan parameter pemicu `notify_before_queue_count` pada cabang; pembakuan numerik NFR (WCAG 2.2 AA, 99.5% SLO); serta penambahan KPI bisnis no-show dan counter-metric kualitas layanan. |

---

## 1. Ringkasan Eksekutif

Quevia adalah platform SaaS multi-tenant untuk membantu bisnis berbasis layanan mengelola jadwal (appointment) dan antrean pelanggan secara digital dan terpadu. Satu aplikasi dipakai banyak perusahaan (tenant), dan setiap perusahaan memiliki cabang, pengguna, layanan, serta data operasional sendiri.

Nilai utama produk adalah **dua sistem yang terintegrasi secara utuh**: Appointment Management (pelanggan memesan jadwal) dan Queue Management (pelanggan check-in, mendapat nomor antrean, dan dipanggil petugas), lengkap dengan layar antrean real-time, notifikasi omnichannel (email & in-app), dan dashboard analitik.

Proyek ini juga dibangun sebagai **portofolio Fullstack Engineer berstandar enterprise**. Domain demo yang dipilih adalah bengkel/service center otomotif, tetapi model datanya dibuat generik agar dapat diadopsi klinik, salon, barbershop, kantor pelayanan publik, dan bisnis layanan lainnya.

Dokumen ini mencakup kebutuhan produk, aturan bisnis, dan contoh struktur database. Rancangan database lengkap (ERD, relasi, constraint, indeks) ada di **dokumen ERD Quevia terpisah**.

Cakupan platform MVP adalah **website responsif** untuk customer, operator, admin, dan layar display. Aplikasi mobile tidak termasuk dalam tahap ini.

---

## 2. Latar Belakang dan Masalah

Pada proses konvensional, pelanggan datang langsung, mengambil nomor fisik, lalu menunggu tanpa kepastian waktu. Masalah yang diselesaikan oleh Quevia:

- Pelanggan tidak tahu perkiraan waktu tunggu dan terpaksa menunggu di lokasi fisik tanpa kepastian.
- Bisnis tidak memiliki visibilitas beban kerja, jam sibuk, dan performa petugas secara real-time.
- Pemesanan manual rawan bentrok jadwal (*double booking*).
- Pelanggan appointment dan walk-in tercampur tanpa aturan prioritas yang terukur.
- Tingkat ketidakhadiran (*no-show*) tinggi akibat ketiadaan pengingat otomatis berjadwal.

---

## 3. Tujuan, Non-Goals, dan Metrik

### 3.1 Tujuan (Goals)

| ID | Tujuan |
| --- | --- |
| G-01 | Pelanggan dapat memesan jadwal layanan secara mandiri tanpa risiko double booking. |
| G-02 | Antrean appointment dan walk-in dikelola dalam satu alur terpadu dengan aturan prioritas yang dapat dikonfigurasi per cabang. |
| G-03 | Perubahan status antrean tampil real-time di dashboard operator, layar display publik, dan aplikasi pelanggan. |
| G-04 | Pelanggan menerima notifikasi otomatis untuk konfirmasi booking, pengingat berjenjang, dan panggilan antrean. |
| G-05 | Pemilik bisnis dapat memantau performa layanan dan efisiensi operasional melalui dashboard analitik. |
| G-06 | Sistem aman untuk multi-tenant: data satu tenant dijamin tidak dapat diakses tenant lain (aplikasi & database RLS). |
| G-07 | Menjadi proyek portofolio yang menunjukkan kompetensi arsitektur frontend, backend, database concurrency, dan real-time processing. |

### 3.2 Non-Goals (di luar MVP)

- Payment gateway dan penagihan langganan sungguhan (cukup simulasi subscription plan).
- Fitur kecerdasan buatan / AI (prediksi jam sibuk, insight otomatis).
- Arsitektur microservices terdistribusi (dipertahankan Modular Monolith).
- Aplikasi mobile (native maupun Flutter); MVP murni berupa website responsif mobile-first.
- Integrasi WhatsApp gateway (pengembangan lanjutan pasca-MVP); reminder dan hasil booking dikirim lewat email dan in-app.

### 3.3 Metrik Keberhasilan (Target & KPIs)

| Metrik | Kategori | Target |
| --- | --- | --- |
| Double booking pada dua request bersamaan untuk slot yang sama | Teknis | 0 (hanya satu request berhasil, lainnya ditolak) |
| Nomor antrean duplikat pada 100 request pembuatan antrean bersamaan | Teknis | 0 |
| Akses data lintas tenant (Cross-tenant leak) | Keamanan | 0 (selalu ditolak 403/404) |
| Propagasi perubahan status ke layar display publik | Performa | < 1 detik (p95, kondisi normal via WebSocket) |
| Respons endpoint cek ketersediaan slot | Performa | < 300 ms (p95, kondisi normal) |
| Cakupan automated test pada skenario kritis (booking, antrean, RLS) | Kualitas | 100% skenario kritis tercakup test otomatis |
| **Tingkat No-Show Pelanggan** (CR-08) | **Bisnis** | **$\le 5\%$** (turun signifikan dari baseline industri ~20%) |
| **Skor Kepuasan Layanan / CSAT (Counter-Metric)** (CR-08) | **Guardrail** | **$\ge 4.5 / 5.0$** (menjaga operator tidak memburu-buru layanan) |

---

## 4. Pengguna dan Hak Akses

Quevia memakai pendekatan multi-tenant dengan lima jenis pengguna:

| Peran | Deskripsi | Kemampuan Utama |
| --- | --- | --- |
| Super Admin | Pengelola platform Quevia | Mengelola tenant, subscription plan, analitik global, dan konfigurasi platform |
| Business Owner / Tenant Admin | Pemilik bisnis yang berlangganan | Mengelola cabang, pegawai, layanan, loket, jam operasional, laporan, aturan antrean |
| Staff / Operator | Petugas yang melayani pelanggan di loket | Check-in, memanggil antrean (*call next*), recall, skip, start service, complete, no-show |
| Customer | Pelanggan bisnis (akun opsional, dapat booking sebagai tamu) | Booking, reschedule, cancel, check-in mandiri, memantau antrean, menerima notifikasi |
| Queue Display | Layar publik di lokasi cabang | Menampilkan nomor yang sedang dipanggil, antrean berikutnya, dan loket secara real-time |

Hierarki data Quevia:

```
Quevia Platform (Global Scope)
├── Business A (tenant)
│   ├── Branch Jakarta (Timezone: Asia/Jakarta)
│   │   ├── Staff Profiles
│   │   ├── Services & Service Counters
│   │   └── Queue Engine (Counters & Tickets)
│   └── Branch Bandung
│       ├── Staff Profiles
│       ├── Services & Service Counters
│       └── Queue Engine
└── Business B (tenant)
    └── Branch Surabaya
        ├── Staff Profiles
        ├── Services & Service Counters
        └── Queue Engine
```

Satu tenant memiliki banyak cabang, dan setiap cabang punya operasional antrean sendiri. Staff hanya boleh melihat antrean dan pelanggan pada cabang tempat ia ditugaskan.

---

## 5. Ruang Lingkup Modul

| Kode | Modul | MVP |
| --- | --- | --- |
| AUTH | Authentication & Organization Management | Ya |
| SVC | Service & Staff Management | Ya |
| APT | Appointment Management | Ya |
| QUE | Queue Management | Ya |
| DSP | Live Queue Display | Ya |
| NTF | Notification Management | Ya (in-app dan email) |
| ANL | Dashboard & Analytics | Ya (dasar) |
| SET | Business Settings | Ya |
| SUB | Subscription & Billing | Opsional (simulasi) |

### Alur Utama Pelanggan
1. Pelanggan membuka website publik Quevia, memilih cabang dan layanan.
2. Memilih tanggal dan jam kedatangan dari slot yang tersedia secara dinamis.
3. Mendapat kode booking unik (contoh: `QV-8K2M4P`) dan email konfirmasi beserta token QR.
4. Datang ke lokasi dan check-in mandiri melalui website, scan QR di ponsel, atau input kode booking; sistem menerbitkan tepat satu tiket antrean.
5. Memantau posisi antrean secara real-time via smartphone dan menerima notifikasi saat giliran mendekati loket.
6. Petugas memanggil, melayani, dan menyelesaikan layanan melalui konsol operator.

Pelanggan walk-in melewati langkah 1–3: petugas atau pelanggan langsung mendaftar di lokasi dan mendapat nomor antrean. Walk-in boleh anonim tanpa kewajiban data pribadi.

---

## 6. Functional Requirements

Prioritas: **M** = Must (MVP), **S** = Should, **C** = Could (pasca-MVP).

### 6.1 AUTH — Authentication & Organization
| ID | Requirement | P |
| --- | --- | --- |
| FR-AUTH-01 | Register, login, dan forgot password untuk owner, staff, dan customer yang memilih membuat akun. | M |
| FR-AUTH-02 | Registrasi organisasi (tenant) beserta cabang pertama. | M |
| FR-AUTH-03 | Manajemen multi-cabang dalam satu tenant. | M |
| FR-AUTH-04 | Undang pengguna ke tenant melalui email. | S |
| FR-AUTH-05 | Role-based access control (RBAC) dengan permission tingkat cabang. | M |
| FR-AUTH-06 | Manajemen profil pengguna. | S |
| FR-AUTH-07 | Customer dapat booking dan check-in sebagai tamu (guest) tanpa membuat akun. | M |

### 6.2 SVC — Service & Staff
| ID | Requirement | P |
| --- | --- | --- |
| FR-SVC-01 | CRUD layanan: nama, deskripsi, estimasi durasi, harga (opsional), kategori, spesialisasi, status ketersediaan. | M |
| FR-SVC-02 | CRUD profil staff dan penugasan ke cabang. | M |
| FR-SVC-03 | Relasi staff dengan layanan yang didukung (`staff_services`). | M |
| FR-SVC-04 | Jam kerja mingguan dan waktu istirahat staff. | M |
| FR-SVC-05 | Jadwal pengecualian: cuti staff, hari libur cabang, perubahan jam kerja khusus. | S |
| FR-SVC-06 | **Proteksi Penonaktifan Staff**: Sistem menolak penonaktifan staff yang masih memiliki janji temu aktif `CONFIRMED`. Admin wajib melakukan *reassign* booking ke staff lain yang berkualifikasi sebelum menonaktifkan. (CR-04) | M |

### 6.3 APT — Appointment
| ID | Requirement | P |
| --- | --- | --- |
| FR-APT-01 | Menghitung slot tersedia berdasarkan jam operasional, istirahat, appointment aktif, durasi layanan, buffer, dan jadwal staff. | M |
| FR-APT-02 | Membuat appointment dengan kode booking unik (prefix `QV-`); booking sebagai tamu mewajibkan email valid. | M |
| FR-APT-03 | Memilih staff secara opsional; jika tidak dipilih, sistem menetapkan staff yang tersedia secara otomatis. | M |
| FR-APT-04 | Mencegah double booking untuk staff dan waktu yang bertabrakan via database exclusion constraint. | M |
| FR-APT-05 | Reschedule appointment sesuai batas waktu `change_cutoff_minutes`. | M |
| FR-APT-06 | Cancel appointment; slot kembali tersedia secara otomatis di database. | M |
| FR-APT-07 | Riwayat appointment pelanggan terdaftar. | M |
| FR-APT-08 | Idempotency pada pembuatan booking agar request ulang tidak membuat data ganda. | M |
| FR-APT-09 | Customer tamu mengakses, reschedule, dan cancel booking dengan kombinasi kode booking dan email yang sama. | M |

### 6.4 QUE — Queue Management
| ID | Requirement | P |
| --- | --- | --- |
| FR-QUE-01 | Registrasi walk-in (boleh anonim) dan pembuatan nomor antrean otomatis. | M |
| FR-QUE-02 | Check-in appointment melalui Web, scan QR, atau kode booking dalam check-in window; mendukung *operator manual check-in override* untuk pelanggan terlambat. (CR-06) | M |
| FR-QUE-03 | Nomor antrean unik per cabang, layanan, dan tanggal; reset harian tanpa menghilangkan riwayat historis. | M |
| FR-QUE-04 | Call next, recall, skip, start service, complete service, cancel; tiket berstatus `SKIPPED` dapat dipanggil kembali (`CALLED`) via recall atau ditutup menjadi `NO_SHOW` di akhir sesi harian. (CR-02) | M |
| FR-QUE-05 | Penanganan no-show otomatis untuk appointment tak hadir dan manual untuk tiket yang dipanggil. | M |
| FR-QUE-06 | Aturan prioritas antrean terkonfigurasi per cabang (`branch_settings`). | S |
| FR-QUE-07 | Riwayat antrean harian per cabang dan per loket. | M |
| FR-QUE-08 | Estimasi waktu tunggu dinamis per tiket antrean. | S |

### 6.5 DSP — Live Queue Display
| ID | Requirement | P |
| --- | --- | --- |
| FR-DSP-01 | Menampilkan nomor yang sedang dipanggil, antrean berikutnya, dan nama loket pelayanan. | M |
| FR-DSP-02 | Update real-time dari konsol operator via WebSocket tanpa refresh browser. | M |
| FR-DSP-03 | Notifikasi audio saat nomor antrean dipanggil (dengan aktivasi izin audio browser satu kali). | S |
| FR-DSP-04 | Mendukung beberapa layar display per cabang. | S |
| FR-DSP-05 | Reconnect otomatis, indikator visual status koneksi, dan sinkronisasi ulang state snapshot setelah koneksi terputus. | M |

### 6.6 NTF — Notification
| Event | Kanal | P |
| --- | --- | --- |
| Booking confirmed | Email | M |
| Appointment reminder (T-24 jam, T-60 menit, T-15 menit check-in) | Email | M |
| Booking rescheduled / cancelled | Email | M |
| Check-in success | In-app | M |
| Queue almost called (dipicu berdasarkan parameter `notify_before_queue_count`) (CR-05) | In-app | S |
| Customer called | In-app dan audio display | M |

Pengiriman notifikasi diproses asynchronous oleh background worker (River), memiliki mekanisme retry dengan exponential backoff, dan tidak pernah memblokir jalur request HTTP.

### 6.7 ANL — Dashboard & Analytics
| ID | Requirement | P |
| --- | --- | --- |
| FR-ANL-01 | Metrik harian: total customer, appointment, walk-in, selesai, dibatalkan, no-show rate. | M |
| FR-ANL-02 | Rata-rata waktu tunggu dan rata-rata durasi pengerjaan layanan. | M |
| FR-ANL-03 | Performa staff dan identifikasi jam sibuk (*peak hours*). | S |
| FR-ANL-04 | Perbandingan performa antar-cabang dan retensi pelanggan. | C |
| FR-ANL-05 | Prediksi beban jam sibuk berbasis AI. | C |

### 6.8 SET dan SUB
- **SET (M)**: Jam operasional, hari libur, aturan antrean, aturan appointment, check-in window, batas no-show, durasi layanan, prioritas antrean, konfigurasi loket, template notifikasi, konfigurasi cabang, dan ambang `notify_before_queue_count`.
- **SUB (C)**: Simulasi plan Free (1 cabang, 3 staff, basic queue), Pro (5 cabang, 20 staff, appointment + analytics), dan Business (unlimited, analytics lanjutan).

---

## 7. User Stories dan Acceptance Criteria

### US-01 Booking Appointment
**Sebagai** customer (terdaftar maupun tamu),  
**Saya ingin** memilih layanan, tanggal, dan jam yang tersedia di cabang tertentu,  
**Agar** saya tidak perlu menunggu lama di lokasi fisik tanpa harus membuat akun.  
**Acceptance Criteria**:
- [ ] Diberikan cabang, layanan, dan tanggal, sistem hanya menampilkan slot waktu yang tidak bertabrakan dengan jadwal kerja, istirahat, libur, dan booking aktif staff.
- [ ] Ketika pelanggan mengonfirmasi booking, sistem membuat record appointment berstatus `CONFIRMED`, menghasilkan `booking_code` unik berformat `QV-XXXXXX`, `qr_token`, dan menjadwalkan email konfirmasi serta 3 reminder via River worker.
- [ ] Ketika dua pelanggan memesan slot dan staff yang sama pada milidetik yang sama, transaksi database kedua digagalkan oleh PostgreSQL exclusion constraint dan mengembalikan `409 SLOT_UNAVAILABLE` tanpa menimbulkan double-booking.

### US-02 Reschedule dan Cancel
**Sebagai** customer atau tamu,  
**Saya ingin** mengubah waktu atau membatalkan booking saya,  
**Agar** jadwal tetap sesuai dengan rencana saya.  
**Acceptance Criteria**:
- [ ] Pelanggan tamu dapat mengakses detail booking melalui verifikasi `booking_code` dan `email` yang identik dengan data pendaftaran.
- [ ] Reschedule hanya diizinkan jika waktu saat ini masih sebelum batas `change_cutoff_minutes` (default 3 jam sebelum jadwal mulai).
- [ ] Ketika booking dibatalkan (`CANCEL`), status berubah menjadi `CANCELLED`, notifikasi reminder yang berstatus `PENDING` otomatis dibatalkan, dan slot waktu lama langsung tersedia untuk pelanggan lain.

### US-03 Check-in Mandiri
**Sebagai** customer yang telah memesan jadwal,  
**Saya ingin** melakukan check-in saat tiba di lokasi (via Web, QR scan, atau kode booking),  
**Agar** saya mendapatkan tiket antrean dan masuk dalam urutan pelayanan loket.  
**Acceptance Criteria**:
- [ ] Check-in hanya diterima di dalam rentang waktu check-in (`check_in_open_before_minutes` s.d. `check_in_close_after_minutes`).
- [ ] Check-in yang berhasil mengubah status appointment menjadi `CHECKED_IN`, menghasilkan tepat satu record `queue_tickets` berstatus `WAITING` dengan nomor antrean terurut, dan mengembalikan `tracking_token`.
- [ ] Upaya check-in berulang untuk appointment yang sama bersifat idempotent dan tidak menghasilkan tiket antrean kedua.

### US-04 Registrasi Walk-in
**Sebagai** staff operator atau pelanggan yang datang langsung,  
**Saya ingin** mendaftarkan antrean walk-in tanpa janji temu sebelumnya,  
**Agar** dapat dilayani sesuai ketersediaan loket.  
**Acceptance Criteria**:
- [ ] Registrasi walk-in dapat dilakukan dengan data pelanggan kosong (anonim).
- [ ] Nomor antrean walk-in di-generate secara terurut per cabang, layanan, dan tanggal menggunakan locking baris counter (`queue_counters`). Tidak ada nomor ganda pada request konkuren.

### US-05 Pemanggilan Antrean (Call Next & Skip)
**Sebagai** operator loket,  
**Saya ingin** memanggil nomor antrean berikutnya yang berhak dilayani,  
**Agar** alur pelayanan berjalan tertib dan teratur.  
**Acceptance Criteria**:
- [ ] Menekan tombol *Call Next* memilih tiket berstatus `WAITING` dengan prioritas tertinggi (appointment ter-check-in diprioritaskan sesuai jadwal, disusul walk-in FIFO) menggunakan `FOR UPDATE SKIP LOCKED`.
- [ ] Status tiket berubah menjadi `CALLED`, mengasosiasikan loket dan staff pemanggil, dan menyiarkan event `queue.ticket_called` ke WebSocket layar display publik < 1 detik.
- [ ] Jika pelanggan tidak hadir saat dipanggil, operator dapat melakukan *Skip* (status menjadi `SKIPPED`). Tiket `SKIPPED` dapat dipanggil kembali (*Recall*) ke status `CALLED` sewaktu-waktu sebelum antrean harian ditutup.

### US-06 Memantau Status Antrean
**Sebagai** pelanggan (terdaftar atau pemegang tiket walk-in),  
**Saya ingin** memantau posisi antrean dan estimasi sisa waktu tunggu di ponsel saya,  
**Agar** saya memiliki kepastian waktu dan dapat menunggu dengan nyaman.  
**Acceptance Criteria**:
- [ ] Halaman pelacakan tiket menampilkan status tiket, jumlah antrean di depan, dan estimasi waktu tunggu secara dinamis.
- [ ] Ketika sisa antrean di depan mencapai ambang `notify_before_queue_count` (default 2), sistem mengirimkan in-app notification `QUEUE_ALMOST_CALLED`.
- [ ] Jika koneksi internet ponsel sempat terputus, antarmuka otomatis reconnect dan menyinkronkan snapshot state terbaru dari REST endpoint.

### US-07 Konfigurasi Bisnis & Cabang
**Sebagai** Tenant Admin,  
**Saya ingin** mengatur jam operasional, layanan, loket, dan aturan batas waktu antrean per cabang,  
**Agar** operasional sistem Quevia sesuai dengan kapasitas dan kebijakan masing-masing cabang.  
**Acceptance Criteria**:
- [ ] Perubahan parameter pada `branch_settings` (seperti `change_cutoff_minutes`, `check_in_close_after_minutes`, `notify_before_queue_count`) langsung berlaku pada transaksi berikutnya tanpa memerlukan restart aplikasi.

### US-08 Isolasi Multi-Tenant
**Sebagai** pemilik bisnis (Tenant Owner),  
**Saya ingin** data pelanggan, transaksi, dan konfigurasi cabang saya terisolasi secara mutlak,  
**Agar** kerahasiaan bisnis saya terjamin dari tenant lain.  
**Acceptance Criteria**:
- [ ] Setiap query aplikasi dibatasi oleh `tenant_id`.
- [ ] Proteksi database PostgreSQL Row-Level Security (RLS) aktif dengan `FORCE ROW LEVEL SECURITY`. Upaya pembacaan atau manipulasi data milik tenant lain selalu mengembalikan penolakan (403/404) di level database.

### US-09 Analitik Performa Bisnis
**Sebagai** Business Owner,  
**Saya ingin** melihat ringkasan metrik harian cabang,  
**Agar** saya dapat mengevaluasi efisiensi pelayanan dan beban antrean.  
**Acceptance Criteria**:
- [ ] Dashboard menampilkan grafik volume pelanggan (appointment vs walk-in), rata-rata waktu tunggu, rata-rata durasi pengerjaan, dan rasio no-show berdasarkan rentang tanggal dan cabang yang dipilih.

### US-10 Penanganan Operasional Khusus (Staff Reassign & Late Override) (CR-04, CR-06)
**Sebagai** Tenant Admin atau Staff Loket,  
**Saya ingin** menangani pembatalan staff mendadak dan pelanggan yang datang terlambat,  
**Agar** kepuasan pelanggan tetap terjaga dalam situasi operasional tak terduga.  
**Acceptance Criteria**:
- [ ] Penonaktifan profil staff divalidasi oleh sistem; jika terdapat janji temu aktif, sistem menampilkan daftar booking yang harus dialihkan (*reassign*) ke staff lain terlebih dahulu.
- [ ] Jika pelanggan appointment datang melewati batas waktu check-in mandiri (>15 menit), staf loket memiliki tombol *Manual Check-in Override* untuk menerbitkan tiket walk-in reguler dan menautkannya ke appointment tersebut.

---

## 8. Aturan Bisnis (Business Rules)

| ID | Aturan |
| --- | --- |
| BR-01 | Semua query dibatasi oleh tenant dan cabang yang berhak diakses pengguna; filter di frontend tidak cukup. Isolasi ditegakkan di database dengan PostgreSQL row-level security (RLS) berdasarkan `tenant_id`. |
| BR-02 | Satu staff tidak boleh memiliki dua appointment aktif dengan rentang waktu yang tumpang tindih (`ex_appointments_no_overlap`). |
| BR-03 | Slot tersedia dihitung dari jam operasional dikurangi waktu istirahat, appointment aktif, libur/cuti, dan buffer waktu antar layanan. |
| BR-04 | Satu appointment menghasilkan maksimal satu queue ticket. **Jika queue ticket yang terhubung dibatalkan (`CANCELLED`) atau ditandai `NO_SHOW`, appointment terkait otomatis beralih status menjadi `CANCELLED` atau `NO_SHOW` dalam satu transaksi** (CR-03). |
| BR-05 | Nomor antrean dibuat di dalam transaksi database dengan row locking (`SELECT ... FOR UPDATE` atau `DO UPDATE` pada counter) per cabang, layanan, dan tanggal; dilarang menggunakan `MAX() + 1`. |
| BR-06 | Nomor antrean di-reset setiap hari; riwayat historis tetap tersimpan melalui kolom `queue_date`. |
| BR-07 | Urutan pemanggilan antrean mengikuti konfigurasi prioritas: appointment ter-check-in dilayani sesuai jendela jamnya, walk-in dilayani secara FIFO, appointment terlambat dapat didegradasi ke prioritas walk-in. |
| BR-08 | Check-in mandiri hanya diterima di dalam check-in window (`check_in_open_before_minutes` s.d. `check_in_close_after_minutes`). |
| BR-09 | Batas waktu reschedule/cancel default 3 jam sebelum jadwal mulai; check-in dibuka 30 menit sebelum dan ditutup 15 menit setelah jadwal mulai; batas toleransi panggilan sebelum no-show adalah 10 menit (semua parameter dapat diatur per cabang). |
| BR-10 | Appointment yang dibatalkan langsung membebaskan slot waktu di database. |
| BR-11 | Operasi kritis yang rawan request berulang (booking, pembuatan tiket antrean, check-in) wajib menyertakan header `Idempotency-Key`. |
| BR-12 | Penghapusan data entitas bisnis menggunakan soft delete (`deleted_at` dan `deleted_by`). Tabel append-only (`audit_logs`, `outbox_events`, `idempotency_keys`) tidak memakai soft delete. |
| BR-13 | Setiap audit log mencatat pelaku aksi; aksi customer tamu dicatat dengan reserved user `guest`, sedangkan proses background job dicatat bernilai null. |
| BR-14 | Estimasi waktu tunggu bersifat dinamis, dihitung dari antrean di depan, jumlah loket aktif, durasi layanan berjalan, dan prioritas tiket. |
| BR-15 | Booking sebagai tamu mewajibkan input alamat email valid untuk pengiriman konfirmasi dan reminder. |
| BR-16 | Akses tamu untuk melihat, reschedule, dan cancel booking diamankan dengan kombinasi kode booking dan email yang sama. |
| BR-17 | Walk-in diperbolehkan anonim tanpa data pelanggan (`customer_id` null). |
| BR-18 | Token QR check-in bersifat unik per appointment dan hanya valid untuk satu kali check-in. |
| **BR-19** | **Penonaktifan profil staff dilarang jika staff masih memiliki appointment aktif `CONFIRMED` di masa depan. Admin wajib melakukan reassign booking terlebih dahulu.** (CR-04) |
| **BR-20** | **Pelanggan appointment yang datang melewati check-in window dapat diterbitkan tiket walk-in oleh staf loket melalui otorisasi *manual check-in override*.** (CR-06) |
| **BR-21** | **Tiket berstatus `SKIPPED` dapat dipanggil kembali (`CALLED`) oleh operator kapan saja selama hari operasional yang sama sebelum penutupan antrean harian.** (CR-02) |

---

## 9. Status dan Transisi Siklus Hidup (State Machines)

### 9.1 Siklus Hidup Appointment
```
(Baru) ──[Booking Berhasil]──> CONFIRMED ──[Check-in Mandiri / Override]──> CHECKED_IN ──[Selesai Layanan]──> COMPLETED
                                  │                                              │
                                  ├──[Cancel Pelanggan / Staff]──> CANCELLED    ├──[Tiket Antrean Dibatalkan]──> CANCELLED (CR-03)
                                  │                                              │
                                  └──[Melewati Batas Window]────> NO_SHOW       └──[Tiket Antrean No-Show]────> NO_SHOW (CR-03)
```

| Status Asal | Status Tujuan | Pemicu / Aksi |
| --- | --- | --- |
| (baru) | CONFIRMED | Pembuatan booking berhasil |
| CONFIRMED | CHECKED_IN | Pelanggan check-in dalam window atau via manual override |
| CONFIRMED | CANCELLED | Dibatalkan oleh pelanggan atau admin |
| CONFIRMED | NO_SHOW | Check-in window tertutup tanpa check-in (background job) |
| CHECKED_IN | COMPLETED | Layanan pada tiket antrean telah diselesaikan oleh operator |
| CHECKED_IN | CANCELLED | Tiket antrean terkait dibatalkan oleh pelanggan/operator (CR-03) |
| CHECKED_IN | NO_SHOW | Tiket antrean terkait ditandai NO_SHOW oleh operator (CR-03) |

### 9.2 Siklus Hidup Queue Ticket
```
(Baru) ──[Walk-in / Check-in]──> WAITING ──[Call Next]──> CALLED ──[Start Service]──> SERVING ──[Complete]──> COMPLETED
                                    │                        │
                                    │                        ├──[Pelanggan Lewat]──> SKIPPED ──[Recall Operator]──> CALLED (CR-02)
                                    │                        │                          │
                                    │                        │                          └──[Tutup Hari]───────────> NO_SHOW (CR-02)
                                    │                        │
                                    │                        └──[Batas Waktu Habis]─> NO_SHOW
                                    │
                                    └──────[Dibatalkan Pelanggan / Operator]─────────> CANCELLED
```

| Status Asal | Status Tujuan | Pemicu / Aksi |
| --- | --- | --- |
| (baru) | WAITING | Pendaftaran walk-in atau check-in appointment |
| WAITING | CALLED | Operator menekan tombol *Call Next* |
| CALLED | SERVING | Operator menekan tombol *Start Service* |
| SERVING | COMPLETED | Operator menekan tombol *Complete Service* |
| CALLED | SKIPPED | Pelanggan tidak segera hadir saat dipanggil (Operator *Skip*) |
| **SKIPPED** | **CALLED** | **Operator memanggil ulang (*Recall*) tiket yang sebelumnya dilewati** (CR-02) |
| **SKIPPED** | **NO_SHOW** | **Tiket tidak hadir hingga akhir sesi hari kerja antrean ditutup** (CR-02) |
| CALLED | NO_SHOW | Pelanggan tidak hadir setelah melebihi `no_show_after_minutes` |
| WAITING / CALLED / SKIPPED | CANCELLED | Dibatalkan oleh pelanggan atau operator |

---

## 10. Non-Functional Requirements (Kuantitatif)

| Area NFR | Kategori | Target Numerik / Tolok Ukur Kepatuhan |
| --- | --- | --- |
| **Aksesibilitas** | Kepatuhan UI | Memenuhi standar **WCAG 2.2 Level AA** pada seluruh halaman publik customer (kontras warna $\ge 4.5:1$, navigasi keyboard penuh, label screen reader). (CR-07) |
| **Ketersediaan** | Keandalan (SLO) | **$\ge 99.5\%$ Uptime** selama jam operasional cabang bisnis. (CR-07) |
| **Pemulihan Bencana** | Keandalan | **RPO $\le 1$ jam** (PostgreSQL WAL backup), **RTO $\le 4$ jam**. (CR-07) |
| **Konkurensi** | Integritas Data | 0 insiden double-booking pada uji beban 50 request konkuren pada slot yang sama; 0 nomor antrean duplikat pada 100 walk-in konkuren. |
| **Latensi Real-time** | Performa | Propagasi event WebSocket ke layar display **p95 < 1.000 ms**. |
| **Latensi API** | Performa | Endpoint ketersediaan slot `availability` **p95 < 300 ms**; endpoint REST standar **p95 < 200 ms**. |
| **Keamanan** | Standar Aplikasi | Kepatuhan OWASP Top 10, hashing password menggunakan **Argon2id**, JWT token bertanda tangan dengan rotasi key, dan isolasi tenant berlapis (Middleware + PostgreSQL RLS). |
| **Ketahanan Antrean** | Operasional | Pola transactional outbox untuk seluruh event; River background jobs memiliki retry otomatis dengan *exponential backoff* dan penanganan *dead-letter*. |
| **Retensi Data** | Penyimpanan | `idempotency_keys` kedaluwarsa dibersihkan setelah 24 jam; `audit_logs` dipertahankan minimal 1 tahun untuk audit kepatuhan. |
| **Bahasa & Lokalisasi** | UX | Bahasa Indonesia sebagai default antarmuka; penanggalan dan jam ditampilkan presisi sesuai zona waktu IANA masing-masing cabang. |

---

## 11. Arsitektur dan Tech Stack Ringkas

Quevia mengadopsi pendekatan **Modular Monolith** dua proses runtime: satu proses API HTTP/WebSocket dan satu proses background Worker.
- **Frontend**: Next.js (App Router) + TypeScript, Tailwind CSS + shadcn/ui, TanStack Query, React Hook Form + Zod.
- **Backend API**: Go + Hertz, REST OpenAPI 3.0, WebSocket (`coder/websocket`).
- **Database**: PostgreSQL (versi stabil terbaru, `btree_gist`, RLS).
- **Cache & Pub/Sub**: Redis (caching referensi, rate limiting, pub/sub event real-time, token allow-list).
- **Worker & Scheduler**: Go + River (PostgreSQL transactional job engine), transactional outbox relay.
- **Observability**: Prometheus, Grafana, OpenTelemetry tracing terstruktur.

---

## 12. Dokumen Terkait

1. **ERD — Quevia Database Design** (`ERD — Quevia Database Design.md`): Rancangan lengkap 26 tabel database, constraint, relasi, indeks, dan RLS policy.
2. **System Architecture & API Contract — Quevia** (`System Architecture & API Contract — Quevia.md`): Arsitektur runtime, diagram alur, kontrak REST endpoint, dan katalog event WebSocket.
3. **PRD Change Log** (`docs/04-prd-change-log.md`): Catatan detail perubahan spesifikasi dan jejak penelusuran (*traceability matrix*).

# Product Improvement Proposal: Quevia

**Tanggal**: 10 November 2026  
**Penulis**: Senior Product Manager & System Analyst  
**Subjek**: Evaluasi Strategi Produk, Cakupan MVP, Prioritas Fitur, dan Proposal Perubahan (Change Proposal)  
**Status**: Phase 3 Deliverable — Completed  

---

## 1. Tinjauan Strategi Produk & Value Proposition

### 1.1 Diferensiasi & Posisi Pasar Quevia
Quevia memiliki proposisi nilai unik (*unique value proposition*) yang membedakannya dari sistem antrean konvensional atau aplikasi appointment standalone:
1. **Unifikasi Dua Dunia (Appointment + Queue)**: Menyatukan pelanggan yang memesan jadwal jauh hari dengan pelanggan walk-in yang datang langsung ke dalam satu antrean berkeadilan (*fair queueing*) dengan prioritas terkonfigurasi.
2. **Frictionless Customer Experience (Guest First)**: Booking dan check-in dapat dilakukan tanpa friksi registrasi akun. Identitas diamankan dengan `booking_code` unik, `qr_token`, dan `tracking_token`.
3. **Database-Enforced Integrity**: Jaminan integritas tidak hanya di level aplikasi, melainkan ditegakkan langsung oleh PostgreSQL melalui `btree_gist` exclusion constraint (anti-double booking) dan Row-Level Security (RLS multi-tenant).
4. **Real-Time Projection**: Arsitektur real-time berbasis snapshot + WebSocket event memberikan kepastian status antrean di layar display publik dan smartphone pelanggan dalam p95 < 1 detik.

### 1.2 Justifikasi Rebranding: QFlow $\rightarrow$ Quevia
- **Diferensiasi Merek**: Nama "QFlow" bersifat terlalu umum dan memiliki banyak homonim di industri perangkat lunak. "Quevia" memberikan identitas brand yang lebih unik, modern, mudah diingat, dan memiliki potensi pendaftaran merek dagang (*trademark*) yang lebih kuat.
- **Konsistensi Visual & Teknis**: Seluruh artefak (PRD, Arsitektur, ERD, kode, schema database, token prefix `QV-`, dan role database `quevia_admin`) diselaraskan ke merek Quevia.

---

## 2. Evaluasi Cakupan MVP (Scope Hygiene)

### 2.1 Fitur yang Dipertahankan dalam MVP (Must-Have)
- **AUTH**: Registrasi tenant, multi-cabang, login berbasis JWT (access token di memori + refresh token di cookie/Redis), dan RBAC berbasis cabang.
- **SVC**: Katalog layanan per cabang, durasi, buffer time, loket pelayanan (*service counters*), dan profil staff.
- **APT**: Perhitungan slot bebas bentrok, booking mandiri tamu/akun dengan idempotency key, reschedule, dan cancel.
- **QUE**: Check-in multi-metode (Web, QR token, booking code), nomor antrean harian per cabang & layanan, call next dengan `FOR UPDATE SKIP LOCKED`, start, complete, skip, dan no-show.
- **DSP**: Layar display antrean kiosk real-time dengan reconnect otomatis dan kontrol audio.
- **NTF**: Email konfirmasi, 3 tingkatan reminder otomatis (T-24h, T-60m, T-15m), dan in-app alert via River background worker.
- **ANL**: Metrik harian mendasar (volume pelanggan, rata-rata waktu tunggu, durasi layanan, tingkat no-show).
- **SET**: Pengaturan jam operasional, check-in window, batas reschedule/cancel, prioritas antrean per cabang.

### 2.2 Fitur yang Dikeluarkan dari MVP (Non-Goals Terkonfirmasi)
Untuk menjaga ketepatan waktu rilis portofolio berkualitas tinggi (8 minggu roadmap):
- *Real Payment Gateway & Billing*: Modul langganan tetap berstatus simulasi/opsional (`SUB`).
- *AI Prediction*: Prediksi waktu sibuk ditiadakan pada MVP; digantikan oleh agregasi data historis langsung.
- *Mobile Apps (iOS/Android)*: Digantikan oleh desain Web Responsif Mobile-First.
- *WhatsApp Notification*: Menghindari ketergantungan biaya provider dan setup BSP kompleks; email menjadi kanal eksternal primer.
- *Microservices*: Ditolak tegas; arsitektur Modular Monolith (Go API + Worker) terbukti paling efisien dan meminimalkan kompleksitas konsistensi data.

---

## 3. Proposal Peningkatan & Registrasi Manajemen Perubahan (Change Management)

Berdasarkan hasil audit Phase 2, berikut adalah daftar usulan perubahan (*Change Proposals*) yang terstruktur:

| Change ID | Requirement Terkait | Kondisi Eksisting | Usulan Perubahan (Proposed Requirement) | Alasan Perubahan (Justification) | Modul Terdampak | Prioritas | Status Usulan |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **CR-01** | Global / Header | Menggunakan nama brand "QFlow", prefix `QF-`, repo `qflow-*`, DB role `qflow_admin`. | Mengubah seluruh referensi brand menjadi **Quevia**, prefix `QV-`, repo `quevia-*`, DB role `quevia_admin`. | Permintaan eksplisit pemilik produk untuk membangun identitas merek yang unik dan konsisten. | PRD, Arch, ERD | **Must** | Disetujui |
| **CR-02** | FR-QUE-04 & Bagian 9.2 | Tiket status `SKIPPED` belum memiliki alur transisi lanjutan yang jelas di PRD. | Menambahkan transisi resmi: tiket `SKIPPED` dapat dipanggil kembali (`CALLED`) via recall operator, atau ditutup menjadi `NO_SHOW` di akhir sesi hari kerja. | Menghilangkan ambang batas limbo pada tiket antrean yang dilewati petugas. | QUE, DSP | **Must** | Disetujui |
| **CR-03** | BR-04 & Bagian 9.1 | Relasi pembatalan tiket terhadap status appointment belum sinkron. | Aturan eksplisit: jika queue ticket yang terhubung dengan appointment dibatalkan (`CANCELLED`) atau `NO_SHOW`, appointment terkait otomatis beralih status menjadi `CANCELLED` atau `NO_SHOW`. | Menjaga konsistensi data antara modul Appointment dan Antrean dalam satu transaksi. | APT, QUE, ERD | **Must** | Disetujui |
| **CR-04** | FR-SVC-02 & BR-02 | Penonaktifan staff tidak memvalidasi janji temu mendatang yang sudah aktif. | Menambahkan validasi: Penonaktifan staff dilarang jika masih memiliki appointment `CONFIRMED`. Admin wajib melakukan *reassign* booking ke staff lain yang berkualifikasi sebelum menonaktifkan. | Mencegah pelanggan datang ke lokasi fisik namun petugasnya tidak ada / akunnya dinonaktifkan. | SVC, APT | **Must** | Disetujui |
| **CR-05** | FR-NTF & SET (ERD `branch_settings`) | Belum ada kolom konfigurasi sensitivitas notifikasi `QUEUE_ALMOST_CALLED` per cabang. | Menambahkan kolom `notify_before_queue_count INTEGER DEFAULT 2` pada tabel `branch_settings` dan konfigurasi modul `SET`. | Memberikan fleksibilitas bagi pemilik cabang dalam mengatur kapan pelanggan harus bersiap mendekati loket. | SET, NTF, ERD | **Should** | Disetujui |
| **CR-06** | FR-QUE-02 & BR-08 | Pelanggan appointment yang datang terlambat (>15 menit) ditolak tanpa opsi pemulihan di lokasi. | Memberikan wewenang kepada Staff/Operator di loket (*manual check-in override*) untuk mengonversi appointment yang kedaluwarsa menjadi tiket antrean walk-in. | Mengurangi gesekan operasional (*customer friction*) saat pelanggan telat karena alasan mendesak namun tetap ingin dilayani. | QUE, Staff UI | **Should** | Disetujui |
| **CR-07** | Bagian 10 (NFR) | NFR masih memuat standar kualitatif ("Website responsif", "Keamanan"). | Menambahkan target numerik spesifik: Kepatuhan aksesibilitas **WCAG 2.2 Level AA**, ketersediaan layanan (*uptime*) **99.5%**, RPO $\le 1$ jam, RTO $\le 4$ jam. | Memenuhi standar *engineering plannability* dan dasar penyusunan Acceptance Testing. | PRD NFR | **Must** | Disetujui |
| **CR-08** | Bagian 3.3 (Metrik) | Metrik hanya berfokus pada target teknis tanpa guardrail operasional. | Menetapkan KPI bisnis: Tingkat no-show $\le 5\%$ (dari baseline industri 20%), dengan *counter-metric*: Skor kepuasan layanan $\ge 4.5/5.0$ (agar operator tidak terburu-buru melayani demi mengejar kuantitas). | Mencegah operator mengorbankan kualitas layanan demi memanipulasi metrik durasi layanan. | PRD Goals | **Should** | Disetujui |

---

## 4. Analisis Dampak Perubahan (Blast Radius Analysis)

1. **Dampak pada PRD**:
   - Pembaruan judul, ringkasan eksekutif, dan seluruh narasi produk menjadi Quevia.
   - Penambahan target metrik bisnis dan guardrail pada Bagian 3.
   - Pembaruan tabel functional requirements (penambahan kontrol reassign staff, override check-in terlambat).
   - Penyempurnaan acceptance criteria pada User Stories (US-01 s.d. US-09).
   - Pembaruan tabel status & transisi (siklus lengkap `SKIPPED`).
   - Pembakuan kuantitatif NFR (WCAG 2.2 AA, 99.5% SLO).
2. **Dampak pada Dokumen Arsitektur & API Contract**:
   - Penyesuaian nama artefak, endpoint prefix, dan role database (`quevia_admin`).
   - Penambahan endpoint `POST /api/v1/appointments/:guid/reassign` (Admin).
   - Dokumentasi penanganan WebSocket banner saat display reconnecting.
3. **Dampak pada Dokumen ERD**:
   - Penambahan kolom `notify_before_queue_count` pada tabel `branch_settings`.
   - Sinkronisasi status constraint pada `appointments` dan `queue_tickets`.

---

## 5. Kesimpulan & Rekomendasi Eksekusi Phase 4

Proposal peningkatan ini mempertahankan seluruh kebutuhan fungsional inti yang telah valid, melengkapi celah operasional kritis yang ditemukan saat audit, dan menetapkan fondasi migrasi merek yang bersih ke **Quevia**.

Semua perubahan telah dianalisis dampaknya dan siap diintegrasikan secara presisi ke dalam dokumen fisik pada **Phase 4: PRD Revision**.

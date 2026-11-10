# Requirement Audit Report: Quevia

**Tanggal**: 10 November 2026  
**Auditor**: Senior Business Analyst & System Analyst (Menggunakan framework `requirements-elicitation`)  
**Subjek**: Audit Kelengkapan Kebutuhan, Ambiguitas, State Machine, dan Edge Cases pada Dokumen Quevia (PRD, Arsitektur, ERD)  
**Status**: Phase 2 Deliverable — Completed  

---

## 1. Executive Summary

Audit ini dilakukan secara sistematis terhadap seluruh dokumen spesifikasi produk Quevia untuk mengidentifikasi kesenjangan (*gaps*), inkonsistensi antar-dokumen, ambiguitas aturan bisnis, dan *edge cases* teknis yang belum tertangani sebelum tahap implementasi rekayasa perangkat lunak dimulai.

Hasil audit menemukan:
- **Tingkat Kematangan Teknis Dasar**: Sangat Tinggi (fondasi konkurensi database, RLS, idempotency, dan arsitektur modular monolith sangat matang).
- **Kesenjangan Kritis (Critical Gaps)**: 4 item (konsistensi migrasi brand QFlow $\rightarrow$ Quevia, transisi siklus tiket `SKIPPED`, penanganan janji temu aktif saat staff dinonaktifkan/cuti mendadak, dan relasi pembatalan tiket terhadap status appointment).
- **Kesenjangan Menengah (Medium Gaps)**: 5 item (ambang batas `QUEUE_ALMOST_CALLED` di konfigurasi cabang, batas kendali operator untuk late check-in override, target numerik formal NFR/WCAG, retensi data audit, dan penanganan timeout/gangguan layar display).
- **Rekomendasi Utama**: Mengonsolidasikan state transition diagram, menambahkan parameter konfigurasi yang hilang pada `branch_settings`, dan membakukan metrik NFR dengan target kuantitatif.

---

## 2. Analisis Dimensi Teknis & Temuan Kesenjangan (Gap Analysis)

### 2.1 Dimensi Data & State Management
| ID | Area / Entitas | Temuan Audit & Kesenjangan | Tingkat Keparahan |
| --- | --- | --- | --- |
| GAP-DATA-01 | State Machine `queue_tickets` | Transisi status tiket dari `SKIPPED` belum terdefinisi secara lengkap pada PRD Bagian 9.2. Tiket yang di-skip oleh operator harus memiliki jalur transisi: dapat dipanggil kembali (`CALLED` via recall) atau kedaluwarsa menjadi `NO_SHOW` jika tidak kunjung hadir hingga akhir hari kerja. | **HIGH** |
| GAP-DATA-02 | Sinkronisasi State `appointments` & `queue_tickets` | Jika pelanggan sudah berstatus `CHECKED_IN` dan memiliki tiket antrean, lalu tiket antrean tersebut dibatalkan (`CANCELLED`) atau ditandai `NO_SHOW` oleh operator, status `appointments` di PRD belum memiliki aturan otomatis apakah ikut menjadi `CANCELLED`/`NO_SHOW` atau tetap `CHECKED_IN`. | **HIGH** |
| GAP-DATA-03 | Parameter Konfigurasi `branch_settings` | Background job `evaluate_queue_proximity` menghasilkan event `QUEUE_ALMOST_CALLED`, namun tabel `branch_settings` di ERD dan PRD belum menyediakan kolom parameter `notify_before_queue_count` (misal default 2 antrean) untuk mengatur sensitivitas pemicu per cabang. | **MEDIUM** |
| GAP-DATA-04 | Soft Delete & Pembersihan Data | Kolom `deleted_at` dan partial index sudah sangat rapi. Namun dokumen belum menyatakan secara eksplisit kebijakan retensi data (*data retention policy*) untuk tabel append-only `audit_logs` dan `outbox_events` yang sudah `PROCESSED`. | **LOW** |

### 2.2 Dimensi Business Logic & Operational Edge Cases
| ID | Skenario Bisnis | Kondisi Uji / Edge Case | Dampak & Risiko | Rekomendasi Solusi |
| --- | --- | --- | --- | --- |
| GAP-BUS-01 | Penonaktifan Staff / Sakit Mendadak | Staff A memiliki 4 janji temu aktif berstatus `CONFIRMED` untuk hari ini, namun tiba-tiba sakit atau status akunnya diubah menjadi `INACTIVE`. | Pelanggan datang namun petugas tidak ada; slot terkunci dan sistem tidak memberi peringatan. | Sistem harus menolak penonaktifan staff jika ada janji temu `CONFIRMED` di masa depan, ATAU mewajibkan Tenant Admin melakukan *reassign* booking ke staff lain yang memiliki kapabilitas `staff_services` serupa. |
| GAP-BUS-02 | Keterlambatan Check-in (>15 Menit) | Pelanggan tiba di lokasi pada menit ke-16 setelah jadwal mulai (melewati `check_in_close_after_minutes`). Status appointment otomatis diubah worker menjadi `NO_SHOW`. | Pelanggan kecewa jika ditolak otomatis padahal antrean di bengkel sedang sepi. | Customer tamu tidak dapat check-in mandiri lewat QR/Web, namun Staff/Operator memiliki hak akses khusus (*manual check-in override*) untuk mengonversi appointment terlambat menjadi tiket antrean walk-in dengan prioritas normal. |
| GAP-BUS-03 | Durasi Layanan Melebihi Estimasi | Pelanggan di Counter 01 mengambil layanan berdurasi estimasi 30 menit, namun pengerjaan fisik memakan waktu 75 menit. | Estimasi waktu tunggu (`estimated_wait_minutes`) pada tiket antrean di belakangnya menjadi tidak akurat. | Algoritma estimasi waktu tunggu harus dinamis: jika layanan sedang berjalan (`SERVING`) melebihi estimasi, sisa durasi dihitung sebagai $\max(5 \text{ menit}, \text{durasi aktual})$ dan di-refresh berkala ke klien status tiket. |
| GAP-BUS-04 | Tiket Antrean Multi-Layanan (*Multi-service*) | Pelanggan membutuhkan dua layanan sekaligus (misal: Ganti Oli + Cuci Kendaraan). | PRD saat ini membatasi 1 tiket antrean hanya memiliki 1 `service_id`. | Menegaskan batasan MVP bahwa 1 tiket = 1 layanan. Jika pelanggan ingin layanan lanjutan, operator menyelesaikan tiket pertama lalu mendaftarkan tiket walk-in baru untuk layanan kedua (*priority transfer* pasca-MVP). |

### 2.3 Dimensi Antarmuka & Real-Time (Interfaces & Real-Time)
| ID | Komponen | Temuan Audit | Rekomendasi Solusi |
| --- | --- | --- | --- |
| GAP-INT-01 | Layar Display Kiosk Kering / Offline | Jika koneksi internet cabang terputus, WebSocket putus. Layar display harus menampilkan indikator visual offline yang ramah dan tidak menampilkan data basi (*stale data*) yang menyesatkan publik. | Layar display menampilkan watermark/banner halus "Menghubungkan kembali..." jika WebSocket terputus > 5 detik; state disinkronkan otomatis saat online. |
| GAP-INT-02 | Kegagalan SMTP Provider Eksternal | Pengiriman email notifikasi diproses oleh River worker dengan retry. Namun jika SMTP provider mati total selama 6 jam, antrean job River menumpuk. | Batasi percobaan maksimum (`max_attempts = 5`), simpan status `FAILED` beserta `last_error`, dan sertakan alert kegagalan pada dashboard platform. |

### 2.4 Dimensi Non-Functional Requirements (NFR)
PRD Bagian 10 saat ini masih memuat pernyataan kualitatif pada beberapa aspek. Berikut hasil audit pembakuan numerik yang direkomendasikan:
1. **Accessibility**: Wajib menetapkan standar minimum **WCAG 2.2 Level AA** untuk antarmuka publik customer (booking, check-in, tracking tiket).
2. **Availability**: Target ketersediaan layanan (*uptime SLO*) $\ge 99.5\%$ pada jam operasional bisnis.
3. **Recovery Point & Recovery Time Objective (RPO/RTO)**: RPO $\le 1$ jam (backup WAL PostgreSQL), RTO $\le 4$ jam.
4. **Performance Targets**:
   - Cek ketersediaan slot: p95 < 300 ms.
   - Panggilan antrean ke WebSocket display: p95 < 1.000 ms.
   - Idempotent booking API: p95 < 500 ms.

---

## 3. Matriks Penilaian Risiko Teknis (Risk Assessment Matrix)

| Kategori Risiko | Deskripsi Risiko | Keparahan | Probabilitas | Rencana Mitigasi |
| --- | --- | --- | --- | --- |
| **Data Integrity** | Inkonsistensi status appointment saat tiket antrean dibatalkan / no-show. | **HIGH** | Sedang | Buat aturan bisnis otomatis: event pembatalan tiket meng-update appointment terkait dalam satu transaksi database. |
| **Operational** | Staff berhalangan hadir saat jadwal appointment sudah penuh. | **HIGH** | Tinggi | Tambahkan validasi pada modul staff dan fitur *reassign appointment* di dashboard admin. |
| **User Experience** | Pelanggan tidak menyadari nomornya hampir dipanggil karena ambang notifikasi tidak fleksibel. | **MEDIUM** | Tinggi | Sediakan kolom `notify_before_queue_count` di `branch_settings`. |
| **Branding / Consistency** | Inkonsistensi nama brand QFlow dan Quevia pada repositori dan kontrak API. | **HIGH** | Tinggi | Eksekusi refaktor penamaan brand komprehensif pada Phase 4 di seluruh 3 file Markdown. |
| **Performance** | Beban WebSocket koneksi kiosk layar display jika banyak cabang membuka koneksi bersamaan. | **LOW** | Rendah | Redis Pub/Sub multiplexing per branch channel (`ws/v1/branches/:guid/queue`). |

---

## 4. Pertanyaan Terarah Berdasarkan Stakeholder (Clarifying Questions)

### Untuk Product Manager / Business Stakeholder:
1. **Skenario Reassign Appointment**: Ketika staff berhalangan hadir atau dinonaktifkan, apakah sistem harus otomatis memindahkan janji temu ke staff lain yang tersedia di cabang tersebut, atau menyerahkan sepenuhnya kepada Tenant Admin untuk memilih manual?
2. **Kebijakan Pelanggan Terlambat**: Apakah staf di loket diizinkan secara resmi untuk menerima pelanggan appointment yang datang terlambat (>15 menit) sebagai tiket walk-in berprioritas reguler?

### Untuk Engineering / System Architect:
1. **Threshold Notifikasi Antrean**: Apakah penambahan kolom `notify_before_queue_count INTEGER DEFAULT 2` pada tabel `branch_settings` di ERD sudah cukup memadai untuk kebutuhan kustomisasi notifikasi per cabang?
2. **State Propagation**: Apakah disepakati bahwa status `appointments` memiliki transisi terikat dengan `queue_tickets` (jika tiket `COMPLETED`, appointment `COMPLETED`; jika tiket `CANCELLED`/`NO_SHOW`, appointment ikut `CANCELLED`/`NO_SHOW`)?

---

## 5. Kesimpulan Audit Phase 2

Audit kebutuhan mengonfirmasi bahwa fondasi arsitektur Quevia sangat kokoh. Kesenjangan yang teridentifikasi tidak memerlukan perombakan arsitektur besar, melainkan penyempurnaan aturan transisi state, penambahan parameter operasional pada konfigurasi cabang, dan pembakuan target numerik NFR. Seluruh temuan ini siap diolah menjadi **Product Improvement Proposal** pada Phase 3.

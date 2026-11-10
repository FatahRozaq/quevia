# PRD Change Log: Quevia

**Tanggal**: 10 November 2026  
**Penulis**: Sarah (Technical Product Owner) & Senior PM / System Analyst  
**Versi Dokumen**: 1.2  
**Status**: Approved & Applied  

---

## 1. Ringkasan Perubahan (Executive Change Summary)

Dokumen ini mencatat seluruh riwayat perubahan spesifikasi dari **QFlow v1.1** menjadi **Quevia v1.2**. Perubahan dilakukan berdasarkan hasil audit kebutuhan (*Requirements Audit* - Phase 2) dan proposal peningkatan produk (*Product Improvement Proposal* - Phase 3) guna memastikan dokumen spesifikasi berstatus *engineering-ready*, konsisten, dan testable.

Penyempurnaan utama mencakup:
1. **Rebranding Menyeluruh**: Perubahan identitas merek dari QFlow menjadi Quevia pada PRD, Arsitektur Sistem, dan ERD.
2. **Penutupan Kesenjangan Siklus State (State Machine Completeness)**: Definisi jalur keluar resmi untuk tiket `SKIPPED` dan sinkronisasi otomatis status `appointments` saat tiket antrean dibatalkan atau ditandai no-show.
3. **Pencegahan Kegagalan Operasional (Operational Resilience)**: Proteksi penonaktifan staff dengan janji temu aktif, prosedur *reassign appointment*, dan izin *operator check-in override* untuk pelanggan terlambat.
4. **Standardisasi Kuantitatif NFR**: Penambahan target numerik aksesibilitas WCAG 2.2 Level AA, ketersediaan layanan 99.5%, serta penetapan KPI bisnis dan *counter-metric* layanan.
5. **Konfigurasi Notifikasi Cabang**: Penambahan parameter `notify_before_queue_count` pada pengaturan cabang.

---

## 2. Tabel Registrasi Manajemen Perubahan (Change Management Ledger)

| Change ID | Requirement Terkait | Requirement Eksisting (v1.1) | Requirement Baru / Revisi (v1.2) | Alasan Perubahan (Rationale) | Modul Terdampak | Status |
| --- | --- | --- | --- | --- | --- | --- |
| **CR-01** | Global Branding | Menggunakan nama brand "QFlow", kode booking `QF-`, direktori `qflow-*`, dan DB role `qflow_admin`. | Menggunakan nama brand **Quevia**, kode booking `QV-`, direktori `quevia-*`, dan DB role `quevia_admin`. | Membangun diferensiasi merek yang lebih kuat, modern, dan unik di pasar SaaS. | PRD, Architecture, ERD | **APPLIED** |
| **CR-02** | FR-QUE-04 & Sec 9.2 | Tiket `SKIPPED` belum memiliki transisi keluar formal di PRD. | Menambahkan transisi: `SKIPPED -> CALLED` (via recall operator) dan `SKIPPED -> NO_SHOW` (saat penutupan antrean harian). | Menghilangkan status limbo pada tiket yang dilewati petugas agar siklus antrean tuntas. | QUE, DSP | **APPLIED** |
| **CR-03** | BR-04 & Sec 9.1 | Relasi pembatalan tiket terhadap status appointment belum sinkron. | Aturan: jika tiket antrean dibatalkan (`CANCELLED`) atau `NO_SHOW`, appointment terkait otomatis menjadi `CANCELLED` atau `NO_SHOW`. | Menjamin konsistensi data dua arah antara janji temu dan tiket antrean. | APT, QUE, ERD | **APPLIED** |
| **CR-04** | FR-SVC-02 & BR-19 | Staff dapat dinonaktifkan tanpa memeriksa jadwal booking aktif di masa depan. | Ditambahkan **FR-SVC-06**: Larangan penonaktifan staff jika memiliki appointment `CONFIRMED`. Wajib *reassign* booking ke staff berkualifikasi lain. | Mencegah pelanggan tiba di lokasi namun staff tidak hadir atau akun dinonaktifkan. | SVC, APT, Admin UI | **APPLIED** |
| **CR-05** | FR-NTF & SET (ERD) | Sensitivitas pemicu notifikasi antrean belum terkonfigurasi di `branch_settings`. | Ditambahkan kolom `notify_before_queue_count INTEGER DEFAULT 2` pada `branch_settings` di ERD dan konfigurasi modul `SET`. | Fleksibilitas konfigurasi jarak antrean pemicu notifikasi *queue proximity* per cabang. | SET, NTF, ERD | **APPLIED** |
| **CR-06** | FR-QUE-02 & BR-20 | Pelanggan terlambat check-in (>15 menit) ditolak permanen tanpa opsi pemulihan di lokasi. | Staff/Operator diberikan akses *manual check-in override* untuk mengonversi appointment terlambat menjadi tiket walk-in. | Mengurangi kekecewaan pelanggan dan memberi diskresi operasional bagi petugas di lokasi. | QUE, Staff UI, APT | **APPLIED** |
| **CR-07** | Sec 10 (NFR) | Standar NFR masih bersifat kualitatif umum ("Website responsif", "Keamanan"). | Ditambahkan target numerik: Kepatuhan **WCAG 2.2 Level AA**, ketersediaan **99.5% SLO**, RPO $\le 1$ jam, RTO $\le 4$ jam. | Memenuhi standar kelayakan rekayasa perangkat lunak (*plannable requirements*). | PRD NFR | **APPLIED** |
| **CR-08** | Sec 3.3 (Metrik) | Metrik hanya berfokus pada target teknis tanpa target hasil bisnis. | Ditambahkan KPI bisnis: Tingkat no-show $\le 5\%$ (target turun dari baseline 20%), dengan *counter-metric* kepuasan $\ge 4.5/5.0$. | Menyelaraskan efisiensi operasional dengan kualitas pengalaman pelanggan. | PRD Goals & Metrics | **APPLIED** |

---

## 3. Analisis Jejak Penelusuran (Traceability Matrix)

| Tujuan Produk (Goal) | Fitur / Requirement Terkait | Metrik Terkait | Acceptance Criteria Terkait |
| --- | --- | --- | --- |
| **G-01**: Booking mandiri anti double-booking | FR-APT-01 s.d. FR-APT-09 | 0 double-booking pada request bersamaan | US-01 (AC-01 s.d. AC-03) |
| **G-02**: Unifikasi antrean dengan prioritas | FR-QUE-01 s.d. FR-QUE-08, FR-SVC-06 | 0 nomor tiket ganda | US-03, US-04, US-05, US-10 |
| **G-03**: Sinkronisasi status real-time | FR-DSP-01 s.d. FR-DSP-05 | Latensi WebSocket p95 < 1 detik | US-05, US-06 |
| **G-04**: Otomatisasi notifikasi | FR-NTF-01 (Email & In-App) | Pengiriman email reminder andal (retry River) | US-01, US-02, US-06 |
| **G-05**: Dashboard analitik bisnis | FR-ANL-01 s.d. FR-ANL-03 | Tingkat no-show $\le 5\%$, CSAT $\ge 4.5/5$ | US-09 |
| **G-06**: Isolasi multi-tenant mutlak | FR-AUTH-01 s.d. FR-AUTH-07, BR-01 | 0 insiden kebocoran lintas tenant | US-08 |
| **G-07**: Portofolio berstandar tinggi | Seluruh modul MVP, NFR kuantitatif | 100% skenario kritis tercakup test otomatis | Seluruh Acceptance Criteria |

---

## 4. Persetujuan & Status Eksekusi
- **Diusulkan oleh**: Tim Audit Kebutuhan & Analis Sistem
- **Disetujui oleh**: Pemilik Produk / Arsitek Perangkat Lunak
- **Status Implementasi Dokumen**: Perubahan telah diterapkan secara penuh pada dokumen spesifikasi Quevia v1.2.

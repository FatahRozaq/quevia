# Changelog

Semua perubahan pada proyek **Quevia** dicatat dalam dokumen ini mengacu pada standar [Semantic Versioning](https://semver.org/) dan [Conventional Commits](https://www.conventionalcommits.org/).

---

## [1.0.0-development] - 2026-11-10

### Ditambahkan
- **Spesifikasi Produk (PRD v1.2)**: Integrasi Appointment Management dan Queue Management dengan model multi-tenant, 10 User Stories, dan 21 Aturan Bisnis.
- **Arsitektur Sistem (v1.1)**: Arsitektur Modular Monolith (Go Hertz API + River Background Worker + Next.js App Router + PostgreSQL + Redis).
- **Rancangan Database (ERD v1.1)**: 26 tabel database dengan PostgreSQL Row-Level Security (`FORCE ROW LEVEL SECURITY`), GiST exclusion constraint (`ex_appointments_no_overlap`), dan row-locking queue counter.
- **Rencana Implementasi Rekayasa (v1.0)**: Jadwal 8 minggu (4 sprint), pemetaan Critical Path, WBS terinci per file, dan Definition of Done.
- **Fondasi Dev Stack**: `docker-compose.yml` untuk PostgreSQL 16, Redis 7, dan Mailpit; `Makefile` shortcut; dan `.env.example`.
- **Standarisasi Kontribusi**: Panduan Git, alur branching (`master`, `development`, `feature/*`), format commit message, dan template Merge Request di `CONTRIBUTING.md`.

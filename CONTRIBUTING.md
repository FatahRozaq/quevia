# Panduan Kontribusi & Git Guideline Quevia

Dokumen ini menetapkan standar penggunaan Git, alur branching, standarisasi commit message, dan proses Merge Request (MR) untuk memastikan konsistensi, meningkatkan kolaborasi, dan memudahkan pemeliharaan kode pada proyek Quevia.

---

## 1. Strategi Branching

| Branch | Deskripsi | Aturan & Kebijakan |
| --- | --- | --- |
| `master` | Cabang stabil untuk *live / production release* (`protected`). | • Hanya boleh di-merge dari `development` & `hotfix/*`.<br>• Production checklist wajib diperiksa sebelum merge.<br>• Menggunakan Git Tags rilis: `v<MAJOR>.<MINOR>.<PATCH>-release`. |
| `development` | Cabang utama integrasi & pengembangan fitur baru (`protected`). | • Sumber awal percabangan untuk seluruh `feature/*`.<br>• Sumber integrasi seluruh fitur sebelum rilis.<br>• Menggunakan Git Tags development: `v<MAJOR>.<MINOR>.<PATCH>-development`. |
| `feature/*` | Cabang pengembangan fitur baru. | • Format: `feature/{modul}/{sub_modul}/{micro_level_modul}`<br>*(contoh: `feature/auth/login`, `feature/appointment/booking`)*.<br>• Digabungkan ke `development` melalui Merge Request (MR). |
| `fix/*` | Cabang perbaikan bug dari suatu feature. | • Format: `fix/{modul}/{sub_modul}/{micro_level_modul}`.<br>• Digabungkan ke `development` melalui Merge Request (MR). |
| `hotfix/*` | Cabang perbaikan darurat langsung di produksi. | • Format: `hotfix/{modul}/{sub_modul}/{micro_level_modul}`.<br>• Wajib dibuat 2 MR: `hotfix → development` dan `hotfix → master`. |

---

## 2. Standarisasi Commit Message

Format pesan commit mengikuti struktur:

```text
<type>(<scope>): <deskripsi singkat> (maksimal 80 karakter)

[Body opsional - penjelasan rincian teknis perubahan]

[Footer opsional]
[Kategori Khusus / Relation Service jika migrasi database]
```

### Kategori `<type>` Wajib:
| Tipe | Deskripsi Penggunaan |
| --- | --- |
| `feat` | Penambahan fitur baru. |
| `fix` | Perbaikan bug / error. |
| `refactor` | Perubahan kode yang bukan bug fixing maupun fitur baru. |
| `docs` | Perubahan khusus dokumentasi (PRD, README, panduan). |
| `test` | Penambahan/pengubahan unit test, concurrency test, atau E2E. |
| `chore` | Perubahan tooling, dependency, konfigurasi build tanpa mengubah logika fitur. |
| `style` | Perubahan format, white-space, indentasi, semicolons. |
| `perf` | Perubahan kode untuk optimasi performa/latensi. |
| `build` | Perubahan pada sistem build atau dependensi eksternal. |
| `ci` | Perubahan konfigurasi CI/CD (GitHub Actions, script deployment). |

### Contoh Pesan Commit:

**Fitur API:**
```text
feat(auth): menyesuaikan login session dan rotasi token

   - Penyesuaian login session berbasis Redis allow-list
   - Rotasi refresh token pada cookie httpOnly saat refresh
   
[API]
```

**Migrasi Database (SQLC):**
```text
chore(queue): penambahan index partial uq_queue_tickets_number

   - Penambahan index unik pada (branch_id, service_id, queue_date, queue_number)
   
[Migration]
queue_service
```

---

## 3. Alur Merge Request (MR) & Pull Request (PR)

1. Buat branch baru dari `development`:
   ```bash
   git checkout development
   git pull origin development
   git checkout -b feature/{modul}/{fitur}
   ```
2. Buat commit terstruktur sesuai panduan di atas.
3. Sebelum membuat MR, pastikan cabang fitur sinkron dengan `development` menggunakan rebase:
   ```bash
   git fetch origin
   git rebase origin/development
   ```
4. Buat Merge Request ke branch `development` menggunakan template berikut:

```markdown
## Deskripsi Perubahan
- [ ] Menambahkan fitur baru
- [ ] Perbaikan bug
- [ ] Pembaruan skema database / migrasi

## Issue / Task Terkait
Closes #[ID_TASK]

## Screenshot / Log Pengujian (Opsional)
[Bukti pengujian lokal atau screenshot UI]

## Checklist
- [ ] Tes lokal berhasil dijalankan (`make test` / `go test`)
- [ ] Tidak ada konflik dengan `development`
- [ ] Dokumentasi diperbarui (jika ada perubahan API/kontrak)
```

---

## 4. Semantic Versioning & Penandaan Tags

Format: `<MAJOR>.<MINOR>.<PATCH>`
- `MAJOR`: Perubahan yang memutus kompatibilitas (*breaking change*).
- `MINOR`: Penambahan fitur baru yang kompatibel ke belakang.
- `PATCH`: Perbaikan bug yang kompatibel ke belakang.

**Penandaan Tag di Git:**
- Untuk rilis di branch `master`:
  ```bash
  git tag -a v1.0.0-release -m "Release: inisialisasi awal platform Quevia"
  git push origin v1.0.0-release
  ```
- Untuk integrasi di branch `development`:
  ```bash
  git tag -a v1.0.0-development -m "Development: integrasi modul awal"
  git push origin v1.0.0-development
  ```

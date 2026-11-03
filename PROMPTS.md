# PROMPTS.md — Perintah Siap Pakai untuk Antigravity

File ini **tidak dibaca otomatis** oleh Antigravity (beda dari `AGENTS.md`
dan `.agents/rules/*.md`). Ini referensi buat kamu copy-paste ke chat
Antigravity. Kalau `PLAN.md` berubah (task digabung/ditambah saat
bimbingan), update file ini juga — jangan biarkan dua-duanya beda isi.

Prinsip tiap prompt: **task mana** (kode, bukan deskripsi panjang) +
**rule/skill mana** (biar tidak ditebak) + **batas** (jangan meluas) +
**bukti** (laporan di akhir, bukan cuma "sudah selesai").

---

## 0. Buka sesi kerja

Pakai salah satu di awal tiap sesi baru.

```
Baca PROGRESS.md. Task mana yang terakhir "Sedang dikerjakan" atau belum
ditandai selesai? Lanjutkan dari situ, jangan mulai task baru dulu.
```

```
Saya mau kerjakan task [kode]. Sebelum mulai, cek PLAN.md apakah
prasyaratnya sudah selesai di PROGRESS.md. Kalau belum, bilang ke saya,
jangan dikerjakan duluan.
```

---

## Fase 0 — Scaffolding

```
Kerjakan Fase 0 PLAN.md (0.1–0.5) berurutan. Ikuti struktur folder di
AGENTS.md §7. Setelah selesai, jalankan go build dan laporkan hasilnya.
```

---

## Fase 1 — Skema data

```
Kerjakan task 1.1–1.6 PLAN.md: model GORM untuk produk, pesanan,
item_pesanan, penyaluran_donasi, profil_pengguna, info_program. Ikuti
.agents/rules/20-data-model.md persis, termasuk aturan integritas. Jangan
buat migrasi dulu, model dulu saja.
```

```
Lanjut task 1.7: tulis migrasi dengan constraint dari 20-data-model.md
(harga>0, nominal>=0, status enum). Lalu task 1.8: seed data uji fiktif.
```

---

## Fase 2 — Auth & role

```
Kerjakan task 2.1: middleware VerifyJWT. Ikuti skill
supabase-jwt-auth-middleware persis, termasuk cara baca SUPABASE_JWT_SECRET
dari env.
```

```
Kerjakan task 2.2: middleware RequireRole. Pastikan 401 vs 403 sesuai
40-api-conventions.md — jangan tertukar.
```

---

## Fase 3 — Produk

```
Kerjakan task 3.1–3.5: endpoint produk (GET list+filter, GET detail,
POST/PATCH/DELETE admin). Ikuti skill fiber-handler-pattern. Pasang
middleware dari Fase 2 di route admin.
```

```
Task 3.6: tambahkan request Postman untuk semua endpoint produk di atas
(sukses + gagal), ikuti skill postman-collection-maintenance. Coba
jalankan.
```

---

## Fase 4 — Pesanan (paling riskan, pecah per task, jangan digabung)

```
Kerjakan task 4.1: POST /api/pesanan. WAJIB ikuti skill gorm-atomic-order
persis — row locking, hitung total dari DB, satu transaksi. Jangan
disederhanakan.
```

```
Kerjakan task 4.3: PATCH status pesanan. Ikuti skill
order-status-lifecycle persis, termasuk tabel transisi yang diizinkan.
```

```
Kerjakan task 4.2 PLAN.md: GET /api/admin/pesanan dengan filter status.
```

```
Kerjakan task 4.4 PLAN.md: GET /api/donasi/ringkasan.
```

```
Kerjakan task 4.5 PLAN.md: POST /api/admin/donasi/penyaluran. Pastikan
validasi nominal tidak melebihi donasi belum tersalurkan (lihat
20-data-model.md aturan #6).
```

```
Kerjakan task 4.6 PLAN.md: GET /api/admin/rekap.
```

```
Task 4.7: tambahkan request Postman P-02 s/d P-07 dan jalankan skenario
T-11 s/d T-14 secara manual. Laporkan mana yang lulus/gagal.
```

---

## Fase 5 — Finalisasi testing

```
Task 5.1: lengkapi koleksi Postman supaya seluruh baris Tabel 4.2
(P-01–P-07) ada requestnya. Cek silang dengan 50-testing-postman.md,
jangan duplikat request yang sudah ada.
```

```
Task 5.3: siapkan environment Postman supaya base_url dan cara dapat
token sudah jelas buat rekan frontend coba. Update README kalau perlu.
Jangan ubah isi collection-nya.
```

(Task 5.2 — catatan hasil pengujian untuk Lampiran 7 — ditulis manual oleh
kamu, bukan oleh Antigravity. Lihat `PLAN.md`.)

---

## Fase 6 — Opsional (Midtrans)

```
Pembimbing sudah menyetujui integrasi Midtrans sandbox. Mulai dari task
6.1 PLAN.md — rencanakan dulu perubahan arsitekturnya (komponen server
tambahan), tunjukkan ke saya sebelum menulis kode.
```

Jangan pakai prompt ini sebelum benar-benar dapat izin pembimbing (lihat
`.agents/rules/00-project-scope.md`).

---

## Commit

```
Task [kode] sudah saya cek manual dan lulus. Buat commit sesuai
60-git-workflow.md (format pesan + kode kebutuhan). Tunjukkan pesan
commit-nya dulu sebelum dieksekusi.
```

---

## Debugging

```
[Tempel error/log di sini]
Jangan langsung rewrite besar. Jelaskan dulu dugaan penyebabnya, cek
apakah melanggar rule di .agents/rules/, baru kasih fix minimal.
```

```
Endpoint [nama] hasilnya [salah seperti apa]. Cek lagi terhadap Definition
of Done di 70-definition-of-done.md — poin mana yang kemungkinan tidak
terpenuhi?
```

---

## Review sebelum demo/sidang

```
Fase 1–5 PLAN.md ditandai selesai di PROGRESS.md. Audit ulang satu per
satu terhadap Definition of Done — jangan percaya catatan lama, cek kode
aslinya. Laporkan task mana yang sebenarnya belum memenuhi semua poin.
```

```
Jalankan ulang seluruh koleksi Postman (Run collection) dan laporkan mana
yang gagal, sebelum saya demo ke pembimbing.
```

---

## Kalau scope berubah saat bimbingan

```
Ada perubahan dari bimbingan: [jelaskan perubahannya].
Update bagian terkait di PLAN.md dan .agents/rules/40-api-conventions.md
saja. Jangan ubah kode dulu sebelum saya konfirmasi rencananya.
```

---

## Guardrail cepat (tempel kapan saja di tengah sesi)

- Agent mulai melebar ke luar scope: **"Stop. Cuma task [kode] dulu, jangan
  sentuh file lain."**
- Mau cek dulu sebelum lanjut: **"Jangan commit dulu. Tunjukkan diff-nya."**
- Hasil meragukan: **"Cek lagi terhadap Definition of Done di
  70-definition-of-done.md poin per poin."**
- Tutup sesi: **"Update PROGRESS.md untuk task yang baru selesai, lalu
  stop."**

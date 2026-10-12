# Portfolio Tracker

Web app ส่วนตัว (ใช้คนเดียว) สำหรับติดตามพอร์ตการลงทุน ย้ายมาจาก `Portfolio-Tracker.xlsx`
ครอบคลุม Brokerage, Mutual Funds, Provident Fund, Savings, Insurance และ Salary & Tax พร้อม Dashboard สรุปภาพรวม

- **Backend**: Go 1.26+ · [Fiber v3](https://gofiber.io) · PostgreSQL 18 ([pgx v5](https://github.com/jackc/pgx))
- **Frontend**: React 19 · Vite 5 · TypeScript · react-router
- **Auth**: username/password (bcrypt) + server-side session cookie (HttpOnly, SameSite=Strict)
- **Deploy**: frontend build แล้ว embed เข้า Go binary ได้ไฟล์ executable ไฟล์เดียว

![dashboard](docs/images/dashboard-light.png)

## Quick start

ต้องมี Go ≥ 1.26 (go.mod pin toolchain 1.27.2 ให้ดาวน์โหลดเอง), Node ≥ 20, Docker (สำหรับ PostgreSQL) และ `make` (Windows ใช้ Git Bash + `scoop install make`)

```bash
cp .env.example .env           # optional: เปลี่ยนรหัสผ่าน DB
make install                   # go mod download + npm ci
make dev-api                   # terminal 1: start DB, migrate, seed, serve :8080
make dev-web                   # terminal 2: Vite :5173 (proxy /api → :8080)
```

หรือทำทีละขั้นเองโดยไม่ใช้ make:

```bash
# 0) PostgreSQL (ครั้งเดียว) — ค่าเริ่มต้น portfolio/portfolio@localhost:5432/portfolio
cp .env.example .env           # optional: เปลี่ยนรหัสผ่าน DB
docker compose up -d

# 1) Backend (terminal 1) — migrate schema + seed user ให้อัตโนมัติ
cd backend
go run ./cmd/server            # http://localhost:8080

# 2) Frontend dev server (terminal 2) — proxy /api → :8080
cd web
npm install
npm run dev                    # http://localhost:5173
```

Login ด้วย **`admin` / `admin`** แล้วระบบจะบังคับให้ตั้งรหัสผ่านใหม่ก่อนใช้งาน

### Production build (ไฟล์เดียว)

```bash
make build                     # = npm run build (→ backend/internal/platform/webui/dist) + go build
./backend/portfolio-tracker    # serve ทั้ง SPA และ API ที่ :8080
```

### Make targets

`make help` แสดงทั้งหมด ที่ใช้บ่อย:

| Target | ทำอะไร |
|---|---|
| `make db-up` / `db-down` | start / stop PostgreSQL (ข้อมูลยังอยู่) |
| `make db-migrate` | apply migration ที่ค้างอยู่ (`go run ./cmd/server migrate`) โดยไม่ start server |
| `make db-reset` | **ลบข้อมูลทั้งหมด** (drop schema `public`) แล้ว migrate ใหม่ ถามยืนยันก่อน, `FORCE=1` ข้ามการถาม |
| `make db-psql` | เปิด psql shell |
| `make seed-sql` | สร้าง `private/seed-data.sql` จาก Excel (`XLSX=...` เปลี่ยนไฟล์ได้) |
| `make test` / `lint` | Go tests / go vet + gofmt + tsc |
| `make vulncheck` | `govulncheck` (Go + stdlib) และ `npm audit` |
| `make check` | lint + test + vulncheck (เหมือน CI) |

### Import ข้อมูลจาก Excel (ครั้งเดียว)

ย้ายข้อมูลจาก `Portfolio-Tracker.xlsx` เข้า DB บน prod:

```bash
make seed-sql XLSX=~/Desktop/Portfolio-Tracker.xlsx   # = cd backend && go run ./cmd/xlsx2sql -in <xlsx> -out ../private/seed-data.sql
./backend/portfolio-tracker migrate                     # schema ต้องมีก่อน
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f private/seed-data.sql
```

- `private/` อยู่ใน `.gitignore` เพราะไฟล์ SQL มีข้อมูลการเงินจริง ห้าม commit
- ทั้งไฟล์อยู่ใน transaction เดียว ถ้าตารางข้อมูลใดมีแถวอยู่แล้ว script จะ abort ทันที จึง import ซ้ำไม่ได้
- ไม่แตะตาราง `users` และ `sessions` ผู้ใช้ `admin` ยัง seed ตัวเองตอน start server ตามเดิม
- import เสร็จแล้วลบ `private/seed-data.sql` ได้เลย

## CI

`.github/workflows/ci.yml` รันทุก push เข้า `main`, ทุก PR, ทุกวันจันทร์ 08:00 (เผื่อมีช่องโหว่ใหม่) และกดรันเองได้:

- **backend**: `make vet`, `make test GOTESTFLAGS=-race` และ migrate บน Postgres 18 ที่สร้างใหม่
- **web**: `npm ci`, typecheck, build
- **build**: build binary ไฟล์เดียวแบบ embed SPA แล้วเก็บเป็น artifact 7 วัน
- **vulncheck**: `make vulncheck`

`.github/dependabot.yml` เปิด PR อัปเดต Go modules, npm และ GitHub Actions ให้อัตโนมัติ

## Configuration

| Env | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | listen address |
| `DATABASE_URL` | `postgres://portfolio:portfolio@localhost:5432/portfolio?sslmode=disable` | ตรงกับ `docker-compose.yml` |
| `SESSION_TTL` | `168h` | อายุ session |
| `COOKIE_SECURE` | `false` | ตั้งเป็น `true` เมื่อ serve ผ่าน HTTPS |
| `CORS_ORIGIN` | – | ใช้เฉพาะตอน frontend อยู่คนละ origin |

ลืมรหัสผ่าน: `docker exec -it portfolio-tracker-db psql -U portfolio -c "DELETE FROM users;"` แล้ว restart backend ระบบจะ seed `admin/admin` ให้ใหม่

Backup: `docker exec portfolio-tracker-db pg_dump -U portfolio -Fc portfolio > backup.dump`

## Project structure

```
backend/
  cmd/server/                  entrypoint (`server` | `server migrate`)
  cmd/xlsx2sql/                Excel → seed SQL แบบใช้ครั้งเดียว (`make seed-sql`)
  internal/domain/             entities, money, validation (ไม่ import อะไรจาก layer อื่น)
  internal/usecase/            auth, generic CRUD, dashboard + ports (interfaces)
  internal/adapter/http/       Fiber routes, DTOs, cookies, error mapping
  internal/adapter/postgres/   repositories, migrations (embedded)
  internal/platform/           config, security (bcrypt, tokens), webui (embedded SPA)
  internal/app/                composition root
web/
  src/components/        Layout, ResourceTable (generic table + inline edit)
  src/pages/             one page per spreadsheet sheet
docs/                    design docs
```

## Design docs

| | |
|---|---|
| [01 · Design pages](docs/01-design-pages.md) | sitemap, wireframes, screenshots, review checklist |
| [02 · Database ER](docs/02-db-er.md) | ER diagram, views (แทนสูตร Excel), Excel → table mapping |
| [03 · User journey](docs/03-user-journey.md) | first login, monthly update, session expiry |
| [04 · System design](docs/04-system-design.md) | architecture, auth flow, API, security, deployment, roadmap |

## Conventions

- เงินเก็บเป็น `BIGINT` และส่งผ่าน API เป็น **integer satang** (`100` = ฿1.00) ส่วนดอกเบี้ยเป็น **basis points** (`250` = 2.5%)
- ค่าที่คำนวณได้อยู่ใน SQL view `v_*` ไม่เก็บซ้ำลงตาราง
- เปลี่ยน schema ด้วยการเพิ่มไฟล์ `backend/internal/adapter/postgres/migrations/NNN_*.sql` ห้ามแก้ไฟล์เดิม

## Security

- Password: bcrypt, ยาว 8–72 bytes, ห้ามเป็น `admin` หรือตรงกับ username; บัญชี `admin/admin` ที่ seed ไว้ต้องเปลี่ยนรหัสก่อนถึงจะเรียก API อื่นได้
- Session: token สุ่ม 256-bit เก็บใน DB เฉพาะ SHA-256; cookie `HttpOnly` + `SameSite=Strict` (+ `Secure` เมื่อ `COOKIE_SECURE=true`); เปลี่ยนรหัสผ่านแล้ว session อื่นถูก revoke หมด
- CSRF: นอกจาก SameSite=Strict แล้ว `POST`/`PUT` ใต้ `/api` ต้องเป็น `Content-Type: application/json` (ส่งข้าม site ไม่ได้ถ้าไม่ผ่าน CORS preflight)
- Rate limit 10 ครั้ง/นาที/IP ที่ `/api/auth/login` และ `/api/auth/password` (เก็บใน memory; ถ้าอยู่หลัง reverse proxy ทุก request จะดูเหมือนมาจาก IP เดียว)
- Headers: helmet + CSP `default-src 'self'` และ HSTS 180 วันเมื่อ `COOKIE_SECURE=true`
- SQL ใช้ parameter ทุกที่, request body ถูก decode แบบ strict (ปฏิเสธ field ที่ไม่รู้จัก) จำกัด 1 MB, error 500 ไม่ส่งรายละเอียดภายในออกไป
- PostgreSQL ใน `docker-compose.yml` bind แค่ `127.0.0.1`
- ข้อมูลการเงิน (`*.xlsx`, `*.csv`, `*.dump`, `backups/`) และ `.env` ถูก git-ignore

ถ้าจะเปิดให้เข้าจากนอกเครื่อง: ให้อยู่หลัง HTTPS (เช่น Caddy / Cloudflare Tunnel) แล้วตั้ง `COOKIE_SECURE=true`

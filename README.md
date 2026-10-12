# Portfolio Tracker

Web app ส่วนตัว (ใช้คนเดียว) สำหรับติดตามพอร์ตการลงทุน ย้ายมาจาก `Portfolio-Tracker.xlsx`
ครอบคลุม Brokerage, Mutual Funds, Provident Fund, Savings, Insurance และ Salary & Tax พร้อม Dashboard สรุปภาพรวม

- **Backend**: Go 1.26+ · [Fiber v3](https://gofiber.io) · PostgreSQL 18 ([pgx v5](https://github.com/jackc/pgx))
- **Frontend**: React 19 · Vite 5 · TypeScript · react-router
- **Auth**: username/password (bcrypt) + server-side session cookie (HttpOnly, SameSite=Strict)
- **Deploy**: frontend build แล้ว embed เข้า Go binary ได้ไฟล์ executable ไฟล์เดียว

![dashboard](docs/images/dashboard-light.png)

## Quick start

ต้องมี Go ≥ 1.26, Node ≥ 20 และ Docker (สำหรับ PostgreSQL)

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
cd web && npm run build        # เขียนไฟล์ลง backend/internal/webui/dist
cd ../backend && go build -o portfolio-tracker ./cmd/server
./portfolio-tracker            # serve ทั้ง SPA และ API ที่ :8080
```

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
  cmd/server/            entrypoint
  internal/config/       env config
  internal/db/           Postgres connect, migrations (embedded), seed admin
  internal/auth/         login, sessions, middleware
  internal/api/          routes, generic CRUD resources, dashboard
  internal/webui/        embedded SPA (web build output)
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
- เปลี่ยน schema ด้วยการเพิ่มไฟล์ `backend/internal/db/migrations/NNN_*.sql` ห้ามแก้ไฟล์เดิม

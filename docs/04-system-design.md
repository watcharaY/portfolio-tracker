# 04 · System Design

## Goals / Non-goals

| Goals | Non-goals (ตอนนี้) |
|---|---|
| ใช้คนเดียว ติดตั้งง่าย: app เป็น binary ไฟล์เดียว + Postgres ใน Docker | multi-user, roles |
| ข้อมูลอยู่ในเครื่องตัวเอง backup ด้วย `pg_dump` | realtime price feed |
| ตัวเลขตรงกับ Excel เดิมทุกสตางค์ | mobile app |
| มี login และ session ที่ปลอดภัยพอสำหรับเปิดผ่าน internet (หลัง HTTPS) | SSO / OAuth |

## Architecture

```mermaid
flowchart LR
    subgraph Browser
        SPA["React 19 + Vite SPA<br/>react-router"]
    end
    subgraph "Go binary (single process)"
        direction TB
        MW["Fiber v3 middleware<br/>recover · logger · helmet · limiter(login)"]
        AUTH["auth.Service<br/>bcrypt · session cookie"]
        API["/api handlers<br/>generic CRUD + dashboard"]
        STATIC["webui (go:embed dist)<br/>SPA fallback → index.html"]
        MW --> AUTH --> API
        MW --> STATIC
    end
    DB[("PostgreSQL 18 (Docker)<br/>tables + v_* views")]
    SPA -- "HTTPS · cookie pt_session" --> MW
    API --> DB
    AUTH --> DB
```

- **Production**: `npm run build` เขียนไฟล์ลง `backend/internal/platform/webui/dist` แล้ว `go build` embed เข้า binary ได้ไฟล์ `.exe` ไฟล์เดียวที่ serve ทั้ง SPA และ API บน origin เดียวกัน (ไม่ต้องตั้ง CORS)
- **Dev**: Vite (:5173) proxy `/api` ไปที่ Go (:8080) ได้ HMR ทั้งฝั่ง frontend

## Repo layout

```
portfolio-tracker/
├─ backend/
│  ├─ cmd/server/main.go           # `server` (serve) | `server migrate`, graceful shutdown
│  └─ internal/
│     ├─ domain/                   # entities, money, validation, domain errors
│     ├─ usecase/                  # auth, generic CRUD, dashboard + ports (interfaces)
│     ├─ adapter/
│     │  ├─ http/                  # Fiber router, DTOs, cookies, error mapping
│     │  └─ postgres/              # repositories, migrate (schema_migrations)
│     │     └─ migrations/001_init.sql
│     ├─ platform/                 # config, security (bcrypt, tokens), webui (go:embed)
│     └─ app/                      # composition root
├─ .github/                        # CI workflow, dependabot
├─ Makefile                        # make help
├─ web/
│  └─ src/
│     ├─ api.ts  auth.tsx  format.ts  hooks.ts
│     ├─ components/ (Layout, ResourceTable)
│     └─ pages/ (Login, ChangePassword, Dashboard, Brokerage, ...)
└─ docs/
```

## Authentication & Session

```mermaid
sequenceDiagram
    participant B as Browser
    participant F as Fiber
    participant S as auth.Service
    participant D as PostgreSQL

    B->>F: POST /api/auth/login {username,password}
    F->>F: limiter (10/min/IP)
    F->>S: Login()
    S->>D: SELECT user by username
    S->>S: bcrypt.Compare (dummy hash ถ้าไม่พบ user → timing เท่ากัน)
    S->>S: token = 32 random bytes (base64url)
    S->>D: INSERT sessions(token_hash = sha256(token), expires_at)
    F-->>B: Set-Cookie pt_session=token; HttpOnly; SameSite=Strict; [Secure]
    Note over B,F: request ต่อไป
    B->>F: GET /api/funds (cookie)
    F->>S: Require(): hash cookie → lookup session ที่ยังไม่หมดอายุ
    alt must_change_password
        F-->>B: 403 PASSWORD_CHANGE_REQUIRED
    else ok
        F-->>B: 200 JSON
    end
```

| การตัดสินใจ | เหตุผล |
|---|---|
| **Server-side session** แทน JWT | revoke ได้ทันที (logout หรือเปลี่ยนรหัสผ่าน) ไม่ต้องทำ refresh token ใช้คนเดียวจึงไม่มีปัญหาเรื่อง scale |
| เก็บ **sha256 ของ token** ใน DB | ถ้าไฟล์ DB หลุด ก็เอา token ไปใช้ไม่ได้ |
| Cookie `HttpOnly` + `SameSite=Strict` | JS อ่าน cookie ไม่ได้ (กัน XSS ขโมย) และ cross-site request ไม่แนบ cookie (กัน CSRF โดยไม่ต้องมี CSRF token เพราะ SPA กับ API อยู่ origin เดียวกัน) |
| `COOKIE_SECURE=true` | ต้องเปิดเมื่อ serve ผ่าน HTTPS |
| bcrypt (cost 10) | มาตรฐาน, เร็วพอสำหรับ login ไม่กี่ครั้ง |
| Seed `admin/admin` + `must_change_password=1` | ตามที่ขอ แต่บังคับเปลี่ยนก่อนเข้าถึงข้อมูล |

## API

ทุก response เป็น JSON และ error อยู่ในรูป `{"error": "..."}` เงินเป็น **satang (int)**

| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| GET | `/api/health` | – | liveness |
| POST | `/api/auth/login` | – | rate limited |
| GET | `/api/auth/me` | session (pending OK) | |
| POST | `/api/auth/logout` | session (pending OK) | |
| POST | `/api/auth/password` | session (pending OK) | `{currentPassword, newPassword}`, rate limited |
| GET | `/api/dashboard` | session | totals + per class + allocation |
| GET | `/api/{resource}` | session | list จาก `v_*` view, filter ได้ด้วย `?field=value` |
| GET | `/api/{resource}/:id` | session | |
| POST | `/api/{resource}` | session | create, ตอบ 201 พร้อม row จาก view |
| PUT | `/api/{resource}/:id` | session | replace ทั้ง row |
| DELETE | `/api/{resource}/:id` | session | 204 |

`{resource}` = `brokers`, `broker-transactions`, `funds`, `pvd-years`, `savings-accounts`, `insurers`,
`insurance-payments`, `salary-months`, `tax-deductions`

**Generic CRUD**: แต่ละ resource ประกาศเป็น spec ใน `internal/adapter/http/resources.go` (table, view, fields + type/enum/required)
ชื่อคอลัมน์ใน SQL มาจาก spec เท่านั้น ส่วนค่าจาก user ส่งผ่าน placeholder `$n` ทั้งหมด จึงไม่มีทาง SQL injection
ถ้าจะเพิ่ม resource ใหม่ ให้เพิ่ม migration + spec 1 ก้อน + page ฝั่ง React 1 ไฟล์

## Security checklist

- [x] Password hash ด้วย bcrypt, ป้องกัน timing attack ตอนหา username
- [x] Session token แบบสุ่ม 256-bit, เก็บเป็น hash, หมดอายุใน 7 วัน, ลบ session ที่หมดอายุแล้วทุกครั้งที่ login
- [x] Rate limit 10/นาที/IP ที่ login และ change password
- [x] Password 8–72 bytes (bcrypt ตัดที่ 72 bytes; ยาวกว่านั้นตอบ 400 แทน 500)
- [x] `helmet` middleware + CSP `default-src 'self'`, `frame-ancestors 'none'`; HSTS เมื่อ `COOKIE_SECURE=true`
- [x] CSRF defense-in-depth: `POST`/`PUT` ใต้ `/api` ต้องเป็น `application/json` (ไม่งั้นตอบ 415)
- [x] Body limit 1 MB, ไม่รับ field ที่ไม่รู้จัก, ตรวจ type และ enum
- [x] `make vulncheck` (govulncheck + npm audit) รันใน CI ทุก push และทุกสัปดาห์; dependabot อัปเดต dependency
- [x] Internal errors ถูก log ไว้ แต่ไม่ส่งรายละเอียดให้ client
- [ ] HTTPS: ถ้าจะเปิดออก internet ให้วางหลัง Caddy / Cloudflare Tunnel / Tailscale และตั้ง `COOKIE_SECURE=true`
- [ ] Backup: ตั้ง scheduled task รัน `pg_dump -Fc` แล้วเก็บไฟล์ไว้ที่อื่น
- [x] Postgres bind ที่ `127.0.0.1` เท่านั้น (ดู `docker-compose.yml`) ควรเปลี่ยนรหัสผ่าน DB ผ่าน `.env`

## Configuration

| Env | Default | |
|---|---|---|
| `ADDR` | `:8080` | listen address |
| `DATABASE_URL` | `postgres://portfolio:portfolio@localhost:5432/portfolio?sslmode=disable` | pgx connection string |
| `SESSION_TTL` | `168h` | Go duration |
| `COOKIE_SECURE` | `false` | `true` เมื่อใช้ HTTPS |
| `CORS_ORIGIN` | (ว่าง) | ใช้เฉพาะตอน SPA อยู่คนละ origin |

## Deployment options

| Option | วิธี | เหมาะกับ |
|---|---|---|
| A. รันบนเครื่องตัวเอง | `docker compose up -d` + `portfolio-tracker.exe` เปิดที่ localhost | ง่ายที่สุด |
| B. Home server + Tailscale | binary + Postgres container บน mini-PC/NAS เปิดผ่าน tailnet | เข้าจากมือถือได้ ไม่เปิด port สู่ internet |
| C. VPS + Caddy | Caddy ทำ TLS อัตโนมัติ, reverse proxy ไป :8080 (หรือใช้ managed Postgres เช่น Neon/Supabase) | เข้าจากที่ไหนก็ได้ (ต้องเปิด `COOKIE_SECURE`) |

## Roadmap

1. Import `Portfolio-Tracker.xlsx` (Go CLI + excelize)
2. Monthly snapshots + กราฟ AUM over time
3. Tax estimator (ขั้นบันไดภาษี + ลดหย่อน)
4. Fund units × NAV (ดึง NAV อัตโนมัติ)
5. Export CSV / backup จากหน้า UI
6. Tests: Go handler tests (httptest + testcontainers-go Postgres), Vitest สำหรับ format/ResourceTable

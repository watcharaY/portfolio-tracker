# Portfolio Tracker

Web app ส่วนตัว (ใช้คนเดียว) สำหรับติดตามพอร์ตการลงทุน ย้ายมาจาก `Portfolio-Tracker.xlsx`
ครอบคลุม Brokerage, Mutual Funds, Provident Fund, Savings, Insurance และ Salary & Tax พร้อม Dashboard สรุปภาพรวม

- **Backend**: Go 1.26+ · [Fiber v3](https://gofiber.io) · PostgreSQL 18 ([pgx v5](https://github.com/jackc/pgx))
- **Frontend**: React 19 · Vite 6 · TypeScript · react-router
- **Auth**: username/password (bcrypt) + server-side session cookie (HttpOnly, SameSite=Strict)
- **Deploy**: frontend กับ backend build และ deploy แยกกัน: Caddy serve SPA และ proxy `/api/*` ไปที่ Go API บน origin เดียวกัน

![dashboard](docs/images/dashboard-light.png)

## ภาพรวมระบบ

### System design

```mermaid
flowchart LR
    U([Owner<br/>browser])
    subgraph web["web · Caddy (web/Dockerfile)"]
        SPA["static SPA<br/>React 19 + Vite build"]
        PX["/api/* → reverse_proxy"]
    end
    subgraph api["api · Go binary (backend/Dockerfile)"]
        direction TB
        HTTP["adapter/http<br/>Fiber v3 · auth guard · DTO · error → status"]
        UC["usecase<br/>AuthService · CRUD · Dashboard"]
        DOM["domain<br/>entities · validation · P/L, %, totals"]
        PG["adapter/postgres<br/>repositories on sqlc · migrations"]
        HTTP --> UC --> DOM
        UC --> PG
    end
    DB[("PostgreSQL 18")]
    U -- "HTTPS · cookie pt_session" --> SPA
    U -- "fetch /api/*" --> PX --> HTTP
    PG --> DB
```

- **แยก FE / BE**: `web` (Caddy) serve ไฟล์ SPA และ proxy `/api/*` ไปที่ `api` บน origin เดียวกัน จึงไม่ต้องตั้ง CORS และ cookie `SameSite=Strict` ใช้ได้ตรงๆ ถ้า `api` ล่ม หน้าเว็บยังโหลดได้ มีแค่ `/api` ที่ตอบ 502
- **Clean architecture ใน backend**: `domain` ไม่ import ชั้นอื่น, `usecase` คุยกับ DB ผ่าน interface (ports), `adapter/postgres` implement ด้วย SQL ใน `queries/*.sql` ที่ sqlc generate เป็น Go, `app` ประกอบทุกอย่างเข้าด้วยกัน
- **การคำนวณ**: SQL รวมยอด (เช่น deposits/withdrawals ต่อ broker, ยอดเบี้ยต่อบริษัท, ลำดับงวด `payment_no`) ส่วนสูตรที่เหลือ (net capital, P/L, %, มูลค่า PVD, ดอกเบี้ยต่อปี, Dashboard) อยู่ใน `domain` ที่เดียว
- **Auth**: session ฝั่ง server, cookie เก็บ token สุ่ม DB เก็บแค่ SHA-256 ของ token, รหัสผ่านเป็น bcrypt ดูรายละเอียดที่ [04 · System design](docs/04-system-design.md)
- **Health**: `/api/health` = process ยังตอบได้, `/api/ready` = ต่อ DB ได้ (503 ถ้าไม่ได้) image `api` มี Docker `HEALTHCHECK` เรียก `server healthcheck` ดูสถานะได้จาก `docker ps`
- **API docs**: OpenAPI อยู่ที่ [`openapi.yaml`](backend/internal/adapter/http/openapi/openapi.yaml) เปิด Swagger UI ได้ที่ `/api/docs` หลัง login (เช่น http://localhost:5173/api/docs) API test ตรวจทุก request/response กับ spec และเช็คว่าทุก route มีใน spec จึงไม่มีทางที่ docs จะไม่ตรงกับ code
- **Dev**: `make dev-web` (Vite :5173) proxy `/api` ไปที่ `make dev-api` (:8080) ส่วน `make app-up` รัน `api` + `web` image เดียวกับ production เบื้องหลังที่ :8000

### User journey

ผู้ใช้มีคนเดียวคือเจ้าของพอร์ต งานหลักคือ **อัปเดตตัวเลขเป็นระยะ แล้วเปิดดูภาพรวมแทนการเปิด Excel**

```mermaid
journey
    title Owner: จากเปิดใช้ครั้งแรก ถึงใช้งานประจำ
    section ครั้งแรก
      Login ด้วย admin/admin: 3: Owner
      ถูกบังคับเปลี่ยนรหัสผ่าน: 4: Owner
      Import ข้อมูลจาก Excel (make db-seed): 4: Owner
    section ทุกเดือน
      อัปเดต Current Value ของ broker: 4: Owner
      อัปเดต Market Value ของกองทุน: 4: Owner
      บันทึกเงินเดือนและ PVD: 4: Owner
      ดู Dashboard (มูลค่ารวม, P/L, สัดส่วน): 5: Owner
    section เมื่อมีเหตุการณ์
      โอนเงินเข้า/ออก broker → เพิ่มรายการ: 4: Owner
      จ่ายเบี้ยประกัน → เพิ่มงวด: 4: Owner
      สิ้นปี → ดู Salary & Tax และลดหย่อน: 3: Owner
```

คะแนน 1–5 คือความลื่นไหลที่คาดไว้

**Login ครั้งแรก**: server สร้าง `admin/admin` ให้เมื่อยังไม่มี user เลย แต่บัญชีนี้ติด `must_change_password` ทุก API ข้อมูลตอบ `403` จนกว่าจะเปลี่ยนรหัสผ่าน

```mermaid
flowchart TD
    A([เปิดเว็บ]) --> B{มี session?}
    B -- ไม่มี / หมดอายุ --> L[/login/]
    L -- POST /api/auth/login --> C{ถูกต้อง?}
    C -- 401 --> L
    C -- ต้องเปลี่ยนรหัส --> CP[/change-password/]
    B -- มี แต่ต้องเปลี่ยนรหัส --> CP
    CP -- POST /api/auth/password --> D{ผ่านกฎ?}
    D -- 400 --> CP
    D -- 200: revoke session อื่นทั้งหมด --> DASH[/Dashboard/]
    B -- มี --> DASH
```

**แก้ข้อมูลรายเดือน**: ทุกตารางใช้ component เดียวกัน (`ResourceTable`) กด Edit แก้ในแถว แล้ว Save ส่ง `PUT` กลับไป ตัวเลขที่คำนวณได้จะมาจาก API เสมอ

```mermaid
sequenceDiagram
    actor O as Owner
    participant UI as React SPA
    participant API as Go API
    participant DB as PostgreSQL
    O->>UI: Edit แถว broker → Current Value
    O->>UI: Save
    UI->>API: PUT /api/brokers/1 {name, current_value, note}
    API->>API: domain.Validate()
    API->>DB: UPDATE brokers … RETURNING id
    UI->>API: GET /api/brokers
    API->>DB: SELECT brokers + SUM(deposit/withdraw)
    API-->>UI: rows + net_capital, unrealised_pl, pl_pct (คำนวณใน domain)
    O->>UI: เปิด Dashboard
    UI->>API: GET /api/dashboard
    API-->>UI: มูลค่า, ต้นทุน, P/L และสัดส่วนของ 5 หมวด
```

flow อื่นและ edge case อยู่ที่ [03 · User journey](docs/03-user-journey.md)

### ER diagram

```mermaid
erDiagram
    users ||--o{ sessions : "login"
    brokers ||--o{ broker_transactions : "deposit / withdraw"
    insurers ||--o{ insurance_payments : "premium"

    users {
        BIGINT id PK
        TEXT username UK
        TEXT password_hash "bcrypt"
        BOOLEAN must_change_password
    }
    sessions {
        TEXT token_hash PK "sha256 ของ cookie"
        BIGINT user_id FK
        TIMESTAMPTZ expires_at
    }
    brokers {
        BIGINT id PK
        TEXT name UK
        BIGINT current_value "สตางค์"
    }
    broker_transactions {
        BIGINT id PK
        BIGINT broker_id FK
        DATE txn_date "nullable"
        TEXT type "deposit | withdraw"
        BIGINT amount "สตางค์ > 0"
    }
    funds {
        BIGINT id PK
        TEXT name UK
        TEXT category "9 หมวด"
        BIGINT cost "สตางค์"
        BIGINT market_value "สตางค์"
    }
    pvd_years {
        BIGINT id PK
        INTEGER year UK
        BIGINT self_contribution
        BIGINT company_contribution
        BIGINT self_profit
        BIGINT company_profit
    }
    savings_accounts {
        BIGINT id PK
        TEXT bank
        BIGINT balance "สตางค์"
        INTEGER interest_bp "basis points"
    }
    insurers {
        BIGINT id PK
        TEXT company UK
    }
    insurance_payments {
        BIGINT id PK
        BIGINT insurer_id FK
        INTEGER year "nullable"
        BIGINT premium "สตางค์ > 0"
    }
    salary_months {
        BIGINT id PK
        INTEGER year "UK (year, month)"
        INTEGER month "1-12"
        BIGINT salary
        BIGINT bonus
        BIGINT withholding_tax
        BIGINT pvd
        BIGINT sso
    }
    tax_deductions {
        BIGINT id PK
        INTEGER year
        TEXT item
        BIGINT amount
    }
```

- **เงินเป็นสตางค์ (`BIGINT`)** ทุก column เงิน 1 บาท = 100 รวมยอดแล้วไม่มีเศษทศนิยมเพี้ยน API ส่งเป็นตัวเลขสตางค์ FE แปลงเป็นบาทตอนแสดง ดอกเบี้ยเป็น basis points (2.5% = 250)
- **ไม่เก็บค่าที่คำนวณได้**: deposits, net capital, P/L, มูลค่า PVD, ดอกเบี้ยต่อปี และยอดใน Dashboard คำนวณทุกครั้งที่อ่าน จึงไม่มีทางที่ตัวเลขจะไม่ตรงกัน
- **ความสัมพันธ์**: มี FK แค่ 3 เส้น (user → session, broker → รายการโอน, บริษัทประกัน → งวดเบี้ย) ทั้งหมด `ON DELETE CASCADE` ลบตัวแม่แล้วรายการลูกหายด้วย UI ถามยืนยันก่อนลบ
- **ใช้คนเดียว**: ตารางข้อมูลไม่มี `user_id` ส่วน `users` / `sessions` มีไว้สำหรับ login เท่านั้น
- ทุกตารางมี `created_at` และ `updated_at` และตารางที่มี `note` ไม่ได้แสดงใน diagram เพื่อให้อ่านง่าย schema เต็มอยู่ที่ [`migrations/`](backend/internal/adapter/postgres/migrations/) และคำอธิบายแต่ละตารางกับ mapping จาก Excel อยู่ที่ [02 · Database ER](docs/02-db-er.md)

## Quick start

ต้องมี Go ≥ 1.26 (go.mod pin toolchain 1.27.2 ให้ดาวน์โหลดเอง), Node ≥ 20, Docker (สำหรับ PostgreSQL) และ `make` (Windows ใช้ Git Bash + `scoop install make`)

```bash
cp .env.example .env           # optional: เปลี่ยนรหัสผ่าน DB
make install                   # go mod download + npm ci
make dev-api                   # terminal 1: start DB, migrate, seed, serve :8080
make dev-web                   # terminal 2: Vite :5173 (proxy /api → :8080)
```

อยากปล่อยรันเบื้องหลังโดยไม่ต้องเปิด terminal ค้างไว้ (ไม่มี hot reload แก้ code แล้วต้อง `make app-up` ใหม่):

```bash
make app-up                    # build + start db, api, web (Caddy) แบบ -d → http://localhost:8000
make app-logs                  # ดู log api + web
make app-down                  # stop api + web (db ยังรันอยู่)
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

### Production build

```bash
make build-web                 # SPA → web/dist
make build                     # API binary → backend/portfolio-tracker (serve แค่ /api)
make run                       # build ทั้งคู่ แล้วรัน binary ด้วย WEB_DIR=../web/dist ที่ :8080
```

### Deploy (Docker)

`deploy/docker-compose.yml` รัน 3 service: `db` (Postgres ไม่เปิด port ออกนอก), `api` (`backend/Dockerfile`) และ `web` (`web/Dockerfile`: Caddy serve SPA และ proxy `/api/*` ไปที่ `api:8080`) แต่ละ image build แยกกันได้ ถ้า API ล่ม หน้าเว็บยังโหลดได้ มีแค่ `/api` ที่ตอบ 502

```bash
cp deploy/.env.example deploy/.env                          # ตั้งรหัสผ่าน DB และ SITE_ADDRESS
make deploy-up                                              # build + start db, api, web
docker compose -f deploy/docker-compose.yml run --rm api migrate   # apply migration (api ก็ migrate เองตอน start)
make deploy-down                                            # stop (ข้อมูลยังอยู่ใน volume)
```

ใช้ domain จริงใน `SITE_ADDRESS` แล้ว Caddy จะออก HTTPS certificate ให้เอง ต้องตั้ง `COOKIE_SECURE=true` ด้วย

### Make targets

`make help` แสดงทั้งหมด ที่ใช้บ่อย:

| Target | ทำอะไร |
|---|---|
| `make app-up` / `app-down` / `app-logs` | รัน api + web เบื้องหลังผ่าน Docker ที่ :8000 (`APP_PORT=...` เปลี่ยนได้) / stop / ดู log |
| `make db-up` / `db-down` | start / stop PostgreSQL (ข้อมูลยังอยู่) |
| `make db-migrate` | apply migration ที่ค้างอยู่ (`go run ./cmd/server migrate`) โดยไม่ start server |
| `make db-reset` | **ลบข้อมูลทั้งหมด** (drop schema `public`) แล้ว migrate ใหม่ ถามยืนยันก่อน, `FORCE=1` ข้ามการถาม |
| `make db-psql` | เปิด psql shell |
| `make seed-sql` | สร้าง `private/seed-data.sql` จาก Excel (`XLSX=...` เปลี่ยนไฟล์ได้) |
| `make db-seed` | `seed-sql` + migrate แล้วโหลดเข้า DB local ตารางต้องว่าง (`make db-reset` ก่อน) |
| `make e2e-seed` | สร้าง DB `portfolio_e2e` ใหม่ด้วยข้อมูลสมมติจาก `backend/testdata/e2e-seed.sql` login `e2e` / `E2e-Tiger-Mango-42` หรือ `admin` / `admin` (ถูกบังคับเปลี่ยนรหัส) |
| `make swagger-ui` | ดึง `swagger-ui-dist` (`SWAGGER_UI_VERSION`) มาไว้ใน `backend/internal/adapter/http/openapi/swagger-ui/` สำหรับ `/api/docs` |
| `make migration name=x` | สร้างไฟล์ migration เปล่า `YYYYMMDDHHMMSS_x.sql` (เวลา UTC) |
| `make sqlc` / `sqlc-check` | generate Go จาก `queries/*.sql` ด้วย sqlc (ผ่าน Docker) / เช็คว่าไฟล์ที่ commit ตรงกับที่ generate ใหม่ |
| `make test` / `lint` | Go tests / sqlc-check + go vet + gofmt + tsc |
| `make e2e` | รัน Playwright ทุก spec ใน `e2e/tests` บน stack แยก (web :8100, DB ในหน่วยความจำ) ด้วยข้อมูลสมมติ แล้วปิด stack ให้ (Windows ใช้ Edge ที่ติดตั้งอยู่) |
| `make docs-screenshots` | สร้างภาพใน `docs/images` ใหม่จากข้อมูลสมมติ (ไม่มีข้อมูลจริงในภาพ) |
| `make e2e-ui` / `e2e-down` | เปิด Playwright UI mode ไว้ดูทีละ step / ปิด stack |
| `make vulncheck` | `govulncheck` (Go + stdlib) และ `npm audit` |
| `make check` | lint + test + vulncheck (เหมือน CI) |
| `make build-web` / `build` / `run` | build SPA / build API binary / รัน binary พร้อม serve `web/dist` |
| `make docker-build` | build image ของ api และ web |
| `make deploy-up` / `deploy-down` | start / stop stack ใน `deploy/` |

### Import ข้อมูลจาก Excel (ครั้งเดียว)

บนเครื่อง local: `make db-reset && make db-seed`

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

- **backend**: `make sqlc-check`, `make vet`, `make test GOTESTFLAGS=-race` และ migrate บน Postgres 18 ที่สร้างใหม่
- **web**: `npm ci`, typecheck, build
- **Docker images**: build `backend/Dockerfile` และ `web/Dockerfile` (ไม่ push)
- **E2E**: build image `api` + `web` แล้วรัน Playwright (Chromium) บนข้อมูลสมมติ ถ้าพังจะแนบ report และ trace เป็น artifact
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
| `TRUSTED_PROXIES` | – | IP/CIDR คั่นด้วย comma (หรือ `private`, `loopback`) ที่อนุญาตให้ส่ง `X-Forwarded-For` มาเป็น IP ของ client ถ้าว่างจะไม่เชื่อ header นี้เลย compose ทุกชุดตั้งเป็น `private` เพราะมีแค่ Caddy ที่คุยกับ api |
| `RATE_LIMIT_PER_MINUTE` | `300` | จำนวน request `/api` ต่อนาทีต่อ IP (ไม่นับ `/api/health`, `/api/ready`) |
| `LOG_FORMAT` | `json` | `json` หรือ `text` (`make` ตั้ง `text` ให้ตอน dev) |
| `WEB_DIR` | – | ถ้าตั้ง จะ serve SPA ที่ build แล้วจากโฟลเดอร์นี้ด้วย (fallback → `index.html`) ถ้าไม่ตั้ง serve แค่ `/api` |

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
  internal/adapter/postgres/   repositories, migrations (embedded), queries/*.sql → internal/gen (sqlc)
  internal/platform/           config, security (bcrypt, tokens)
  internal/app/                composition root
  Dockerfile                   API image (distroless)
web/
  Caddyfile, Dockerfile  web image: SPA + proxy /api
  src/components/        Layout, ResourceTable (generic table + inline edit)
  src/pages/             one page per spreadsheet sheet
deploy/                  docker-compose.yml + .env.example สำหรับ deploy
docs/                    design docs
```

## Design docs

| | |
|---|---|
| [01 · Design pages](docs/01-design-pages.md) | sitemap, wireframes, screenshots |
| [02 · Database ER](docs/02-db-er.md) | ER diagram, conventions, Excel → table mapping |
| [03 · User journey](docs/03-user-journey.md) | first login, monthly update, session expiry |
| [04 · System design](docs/04-system-design.md) | architecture, auth flow, API, security, deployment, roadmap |

## Conventions

- เงินเก็บเป็น `BIGINT` และส่งผ่าน API เป็น **integer satang** (`100` = ฿1.00) ส่วนดอกเบี้ยเป็น **basis points** (`250` = 2.5%)
- ค่าที่คำนวณได้ไม่เก็บลงตาราง: SQL รวมยอดใน `queries/*.sql` ส่วนสูตรอยู่ใน `backend/internal/domain`
- เปลี่ยน schema ด้วย `make migration name=x` แล้วแก้ไฟล์ใหม่ ห้ามแก้ไฟล์เดิม จากนั้น `make db-migrate` และ `make sqlc`
- เพิ่มหรือแก้ SQL ใน `backend/internal/adapter/postgres/queries/*.sql` แล้วรัน `make sqlc` และ commit `internal/gen` ไปด้วย

## Security

- Password: bcrypt, ยาว 8–72 bytes, ห้ามเป็น `admin` หรือตรงกับ username; บัญชี `admin/admin` ที่ seed ไว้ต้องเปลี่ยนรหัสก่อนถึงจะเรียก API อื่นได้
- Session: token สุ่ม 256-bit เก็บใน DB เฉพาะ SHA-256; cookie `HttpOnly` + `SameSite=Strict` (+ `Secure` เมื่อ `COOKIE_SECURE=true`); เปลี่ยนรหัสผ่านแล้ว session อื่นถูก revoke หมด
- CSRF: นอกจาก SameSite=Strict แล้ว `POST`/`PUT` ใต้ `/api` ต้องเป็น `Content-Type: application/json` (ส่งข้าม site ไม่ได้ถ้าไม่ผ่าน CORS preflight)
- Rate limit ต่อ IP (in-memory): ทุก `/api` 300 ครั้ง/นาที (`RATE_LIMIT_PER_MINUTE`) และ `/api/auth/login`, `/api/auth/password` 10 ครั้ง/นาที เกินแล้วตอบ `429` พร้อม `Retry-After` และ `X-RateLimit-*`
- Client IP: เชื่อ `X-Forwarded-For` เฉพาะเมื่อ request มาจาก `TRUSTED_PROXIES` (Caddy) จึงปลอม IP จากภายนอกไม่ได้
- Log: JSON หนึ่งบรรทัดต่อ request (`request_id`, method, route, status, latency, IP, user agent, user id) ไม่มี body, cookie หรือ query string ทุก response มี `X-Request-ID` (รับจาก upstream ถ้าเป็น ASCII ที่อ่านได้ ไม่งั้นสร้างใหม่)
- API ส่ง `Cache-Control: no-store` ทุก response
- Headers: helmet + CSP `default-src 'self'` และ HSTS 180 วันเมื่อ `COOKIE_SECURE=true`
- SQL ใช้ parameter ทุกที่, request body ถูก decode แบบ strict (ปฏิเสธ field ที่ไม่รู้จัก) จำกัด 1 MB, error 500 ไม่ส่งรายละเอียดภายในออกไป
- PostgreSQL ใน `docker-compose.yml` bind แค่ `127.0.0.1`
- ข้อมูลการเงิน (`*.xlsx`, `*.csv`, `*.dump`, `backups/`) และ `.env` ถูก git-ignore

ถ้าจะเปิดให้เข้าจากนอกเครื่อง: ให้อยู่หลัง HTTPS (เช่น Caddy / Cloudflare Tunnel) แล้วตั้ง `COOKIE_SECURE=true`

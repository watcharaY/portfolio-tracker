# 01 · Design Pages (Review)

เอกสารนี้ใช้ review หน้าจอทั้งหมด แต่ละหน้า map มาจากชีตใน `Portfolio-Tracker.xlsx` 1:1
เพื่อให้ย้ายจาก Excel มาใช้ได้ทันที แล้วค่อยปรับ UX ทีหลัง

## Sitemap

```mermaid
flowchart LR
    Login["/login"] -->|admin/admin ครั้งแรก| CP["/change-password"]
    Login -->|รหัสผ่านถูกเปลี่ยนแล้ว| Dash
    CP --> Dash["/ (Dashboard)"]
    Dash --- B["/brokerage"]
    Dash --- F["/funds"]
    Dash --- P["/pvd"]
    Dash --- S["/savings"]
    Dash --- I["/insurance"]
    Dash --- T["/salary"]
```

| Route | หน้า | ชีตต้นทาง | API |
|---|---|---|---|
| `/login` | Login | – | `POST /api/auth/login` |
| `/change-password` | ตั้งรหัสผ่านใหม่ (บังคับครั้งแรก) | – | `POST /api/auth/password` |
| `/` | Dashboard | Dashboard | `GET /api/dashboard` |
| `/brokerage` | Brokers + Deposit/Withdraw log | Brokerage | `/api/brokers`, `/api/broker-transactions` |
| `/funds` | Mutual Funds | Funds | `/api/funds` |
| `/pvd` | Provident Fund รายปี | PVD | `/api/pvd-years` |
| `/savings` | Savings Accounts | Savings | `/api/savings-accounts` |
| `/insurance` | Companies + Premium log | Insurance | `/api/insurers`, `/api/insurance-payments` |
| `/salary` | Salary & Tax (เลือกปี) | Salary & Tax 2026 | `/api/salary-months`, `/api/tax-deductions` |

## App shell

```
┌────────────────┬──────────────────────────────────────────────┐
│ Portfolio      │  <Page title>                                │
│ Tracker        │  hint text                                   │
│                │ ┌──────────────────────────────────────────┐ │
│ ▸ Dashboard    │ │ Card                              [+Add] │ │
│   Brokerage    │ │ table ...                                │ │
│   Mutual Funds │ └──────────────────────────────────────────┘ │
│   Provident    │                                              │
│   Savings      │                                              │
│   Insurance    │                                              │
│   Salary & Tax │                                              │
│                │                                              │
│ admin [Logout] │                                              │
└────────────────┴──────────────────────────────────────────────┘
  มือถือ (< 760px): sidebar กลายเป็นแถบเมนูแนวนอนด้านบน
```

- Light/Dark ตาม OS (`prefers-color-scheme`) และ override ได้ด้วย `<html data-theme="dark|light">`
- ตัวเลขชิดขวา และใช้ `tabular-nums` ทุกตาราง

## 1. Login

![login](images/login-dark.png)

- ฟอร์มมีแค่ Username/Password ไม่มี "remember me" เพราะ session อยู่ได้ 7 วันอยู่แล้ว
- ถ้าผิด แสดงข้อความกลาง ๆ ว่า `invalid username or password` (ไม่บอกว่าผิดที่ user หรือ password)
- จำกัด 10 ครั้ง/นาที/IP (Fiber limiter)

## 2. Change password (บังคับครั้งแรก)

```
┌──────── ตั้งรหัสผ่านใหม่ ────────┐
│ ⚠ บัญชีนี้ยังใช้รหัสผ่านเริ่มต้น   │
│ Current password  [........]   │
│ New password      [........]   │
│ Confirm           [........]   │
│ [ Save password ]   Log out    │
└────────────────────────────────┘
```

- ถ้ายังไม่ได้เปลี่ยน backend ตอบ `403 PASSWORD_CHANGE_REQUIRED` ทุก endpoint ยกเว้น `/auth/me`, `/auth/logout`, `/auth/password`
- รหัสผ่านใหม่ต้องยาวอย่างน้อย 8 ตัวอักษรและห้ามเป็น `admin` เปลี่ยนแล้ว session อื่นทั้งหมดจะถูก revoke

## 3. Dashboard

![dashboard light](images/dashboard-light.png)

![dashboard dark](images/dashboard-dark.png)

| ส่วน | ที่มา (เทียบสูตรใน Excel) |
|---|---|
| KPI: AUM / Total cost / Unrealised P/L / P/L % | `Dashboard!B5:E5` |
| Est. savings interest | `SUM(balance × rate)` ⇒ `Dashboard!B6` |
| Allocation bar | `Dashboard!G9:G13` แสดงเป็น stacked bar 100% แบบแท่งเดียว hover แล้วเห็นรายละเอียด |
| ตาราง By asset class | `Dashboard!B8:G15` |

แต่ละ asset class มีสีประจำตัวคงที่ (Stocks=น้ำเงิน, Funds=ส้ม, PVD=เขียวน้ำทะเล, Savings=เหลือง, Insurance=ชมพู)
สีไม่เปลี่ยนตามลำดับหรือการกรอง ข้อมูลเดียวกันมีในตารางด้วย จึงไม่ต้องดูจากสีอย่างเดียว

## 4. Brokerage

![brokerage](images/brokerage-light.png)

- Card บน = **Brokers** แก้ได้เฉพาะ Name / Current Value / Note ส่วน Deposits, Withdrawals, Net Capital, P/L และ P/L % มาจาก view `v_brokers`
- Card ล่าง = **Deposit / Withdraw log** เลือก Broker จาก dropdown และ Type ได้แค่ deposit/withdraw (เท่ากับ data validation ใน Excel)
- เพิ่มหรือลบ log แล้ว card บนจะ refresh เอง

## 5. Mutual Funds / 6. PVD / 7. Savings

ใช้ component `ResourceTable` ตัวเดียวกันทั้งหมด

```
┌ Funds ───────────────────────────────────────────────── [+ Add] ┐
│ Fund │ Category ▾ │ Cost │ Market Value │ P/L (auto) │ P/L % │ Note │ Edit Delete │
│ ...                                                            │
│ Total│            │ Σ    │ Σ            │ Σ          │       │      │             │
└────────────────────────────────────────────────────────────────┘
```

- คอลัมน์ `auto` = คำนวณจาก view แก้ไม่ได้
- Savings: กรอก Interest Rate เป็น % (2.5) แต่เก็บใน DB เป็น basis points (250)

## 8. Insurance

![insurance](images/insurance-light.png)

- Companies: Payments (นับงวด) และ Total Paid คำนวณให้อัตโนมัติ
- Premium log: `Payment #` ได้จาก `ROW_NUMBER() OVER (PARTITION BY insurer)` เหมือน `COUNTIF($F$6:$F6,$F6)` ใน Excel

## 9. Salary & Tax

```
Salary & Tax                                   Year [2026]
┌ Monthly income 2026 ─────────────────────────── [+ Add] ┐
│ Year │ Month ▾ │ Salary │ Bonus │ WHT │ PVD │ SSO │       │
└──────────────────────────────────────────────────────────┘
┌ Deductions 2026 ─────────────────────────────── [+ Add] ┐
│ Year │ Item │ Amount │                                   │
└──────────────────────────────────────────────────────────┘
```

- ใน Excel ชีตนี้ผูกกับปี 2026 ใน app เปลี่ยนเป็นตัวเลือกปี (`?year=`) จึงใช้ได้ทุกปี
- คอลัมน์ Salary Inc. / Inc. % / "if not prorated" ใน Excel ยังไม่ได้ย้ายมา ดู Open questions

## Review checklist / Open questions

- [ ] **Salary Inc. %, "if not prorated"**: ต้องการคำนวณใน view หรือแสดงเป็นรายงานแยก?
- [ ] **Tax estimate**: ใช้ข้อมูลเงินเดือนและรายการลดหย่อนที่มีอยู่แล้ว คำนวณภาษีทั้งปีตามขั้นบันได และเทียบกับภาษีที่ถูกหัก ณ ที่จ่าย?
- [ ] **ประวัติมูลค่า (snapshots)**: เก็บ AUM รายเดือนเพื่อทำกราฟ trend? (ร่างตารางไว้ใน 02-db-er.md)
- [ ] **Import จาก Excel**: ทำ CLI สำหรับ import `Portfolio-Tracker.xlsx` ครั้งเดียว?
- [ ] **Fund units/NAV**: ตอนนี้กรอก market value เอง อนาคตจะเก็บจำนวน units × NAV แล้วดึง NAV อัตโนมัติไหม?

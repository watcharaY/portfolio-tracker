# 03 · User Journey

ผู้ใช้มีคนเดียวคือเจ้าของพอร์ต (role: owner/admin) งานหลักคือ **อัปเดตตัวเลขเป็นระยะ แล้วเปิดดูภาพรวม**
แทนการเปิด Excel

## Journey overview

```mermaid
journey
    title Owner — จากติดตั้งครั้งแรก จนถึงใช้งานประจำ
    section First run
      Start server (seed admin/admin): 3: Owner
      Login ด้วย admin/admin: 3: Owner
      ถูกบังคับเปลี่ยนรหัสผ่าน: 4: Owner
    section Setup data
      เพิ่ม brokers / funds / savings / insurers: 3: Owner
      กรอก deposit/withdraw log และเบี้ยประกันย้อนหลัง: 2: Owner
    section Routine (รายเดือน)
      Login: 5: Owner
      อัปเดต Current Value ของ broker: 4: Owner
      อัปเดต Market Value ของกองทุน: 4: Owner
      บันทึกเงินเดือนและ PVD เดือนนี้: 4: Owner
      ดู Dashboard (AUM, P/L, allocation): 5: Owner
    section Event-based
      โอนเงินเข้า/ออก broker → เพิ่ม log: 4: Owner
      จ่ายเบี้ยประกัน → เพิ่ม premium: 4: Owner
      สิ้นปี → ดู Salary & Tax + ลดหย่อน: 3: Owner
```

คะแนน 1–5 = ความลื่นไหลที่คาดไว้ ส่วนที่ได้ 2 คือการกรอกข้อมูลย้อนหลัง ซึ่งเป็นเหตุผลที่ควรมีตัว import จาก Excel (ดู Open questions ใน 01)

## Flow 1 — First login (seeded admin/admin)

```mermaid
flowchart TD
    A([เปิด http://localhost:8080]) --> B{มี session cookie?}
    B -- ไม่มี --> L[/login/]
    B -- มี --> M[GET /api/auth/me]
    M -- 401 --> L
    L --> C[กรอก admin / admin]
    C --> D[POST /api/auth/login]
    D -- 401 --> L
    D -- 200 mustChangePassword=true --> CP[/change-password/]
    CP --> E[POST /api/auth/password]
    E -- 400 weak / wrong current --> CP
    E -- 200 --> R[revoke session อื่นทั้งหมด] --> DASH[/Dashboard/]
    M -- 200 mustChangePassword=true --> CP
    M -- 200 --> DASH
```

## Flow 2 — Monthly update

```mermaid
sequenceDiagram
    actor O as Owner
    participant UI as React SPA
    participant API as Go Fiber
    participant DB as PostgreSQL

    O->>UI: เปิด /brokerage
    UI->>API: GET /api/brokers
    API->>DB: SELECT * FROM v_brokers
    DB-->>API: rows (+computed)
    API-->>UI: JSON
    O->>UI: Edit Broker A → Current Value = 100,000.00
    UI->>API: PATCH /api/brokers/1 {current_value: 10000000}
    API->>DB: UPDATE brokers SET ... , updated_at=now
    API-->>UI: row ใหม่จาก v_brokers (P/L อัปเดตแล้ว)
    O->>UI: เปิด Dashboard
    UI->>API: GET /api/dashboard
    API-->>UI: totals + allocation
```

## Flow 3 — Session expiry

1. Cookie หมดอายุ (7 วัน) หรือ logout จากที่อื่น
2. API ใดก็ตามตอบ `401` แล้ว `api.ts` ยิง event `auth:expired`
3. `AuthProvider` set user = null ทำให้ `RequireAuth` redirect ไป `/login` พร้อม `state.from`
4. Login สำเร็จแล้วพากลับไปหน้าเดิม

## Edge cases ที่รองรับแล้ว

| กรณี | พฤติกรรม |
|---|---|
| เพิ่ม broker ชื่อซ้ำ | `409 a record with this value already exists` |
| Type ไม่ใช่ deposit/withdraw | `400 type must be one of [deposit withdraw]` |
| ลบ broker ที่มี log อยู่ | log ถูกลบไปด้วย (cascade) และ UI ถามยืนยันก่อน |
| Brute-force login | 10 ครั้ง/นาที/IP แล้วตอบ `429` |
| ยังไม่เปลี่ยนรหัสผ่าน แต่ยิง API ตรง | `403 PASSWORD_CHANGE_REQUIRED` |

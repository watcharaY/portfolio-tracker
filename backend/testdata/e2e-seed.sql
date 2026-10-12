-- Made-up fixture data for API and browser tests. Every amount is in satang.
-- Expected dashboard totals live in e2e-expected.json; keep the two files in step.
-- The script empties every table first, so it can be re-run before each test file.

DO $$
BEGIN
    IF current_database() !~ '_(e2e|test)$' THEN
        RAISE EXCEPTION 'e2e seed refused: database % does not end in _e2e or _test', current_database();
    END IF;
END $$;

BEGIN;

TRUNCATE users, sessions, brokers, broker_transactions, funds, pvd_years, savings_accounts,
         insurers, insurance_payments, salary_months, tax_deductions RESTART IDENTITY CASCADE;

-- 1: admin/admin, still forced to change the password.
-- 2: e2e / E2e-Tiger-Mango-42, password already changed.
INSERT INTO users (username, password_hash, must_change_password) VALUES
    ('admin', '$2a$10$uprEzWnQErBBN8xbUrzv3O5legD6GTlGCyR1XSRECu8OuGHqp0x3y', TRUE),
    ('e2e',   '$2a$10$sULXD35JYAKN3n4xtDLBzeq7IUdUKlm7Y/SjNvYJkZILhOzg1eBY.', FALSE);

-- Alpha: net capital 1,100,000.00, value 1,200,000.00, P/L +100,000.00.
-- Beta:  deposits equal withdrawals, so net capital is 0 and P/L % is empty.
INSERT INTO brokers (name, current_value, note) VALUES
    ('Broker Alpha', 120000000, ''),
    ('Broker Beta',  0,         'ปิดบัญชีแล้ว');

INSERT INTO broker_transactions (broker_id, txn_date, type, amount, note) VALUES
    (1, '2025-01-15', 'deposit',  100000000, 'initial'),
    (1, '2025-06-01', 'deposit',   20000000, ''),
    (1, '2025-09-01', 'withdraw',  10000000, 'ถอนบางส่วน'),
    (2, NULL,         'deposit',    5000000, 'no date'),
    (2, '2026-02-01', 'withdraw',   5000000, '');

-- One fund per category: gains, losses, zero P/L and a satang amount.
INSERT INTO funds (name, category, cost, market_value, note) VALUES
    ('Fund Equity A',   'Equity',       10000000, 12000000, ''),
    ('Fund Bond B',     'Fixed Income', 20000000, 20400000, ''),
    ('Fund Cash C',     'Money Market',  5000000,  5050050, 'satang'),
    ('Fund Mixed D',    'Mixed',         8000000,  7600000, 'loss'),
    ('Fund Gold E',     'Gold',          3000000,  3600000, ''),
    ('Fund SSF F',      'SSF',          10000000, 11000000, ''),
    ('Fund RMF G',      'RMF',          15000000, 16500000, ''),
    ('Fund ThaiESG H',  'ThaiESG',      10000000,  9500000, 'loss'),
    ('กองทุนอื่น I',      'Other',         1000000,  1000000, 'P/L เป็นศูนย์');

-- 2025 gains, 2026 losses.
INSERT INTO pvd_years (year, self_contribution, company_contribution, self_profit, company_profit, fund_allocation) VALUES
    (2025, 6000000, 3000000, 300000,  150000, 'Equity 60 / Bond 40'),
    (2026, 3600000, 1800000, -120000, -60000, 'Equity 50 / Bond 50');

INSERT INTO savings_accounts (bank, balance, interest_bp, note) VALUES
    ('ธนาคาร ก',  50000000, 150, ''),
    ('Bank B',   30000000, 250, ''),
    ('Bank C',          0,   0, 'empty');

-- Insurer Three has no payments.
INSERT INTO insurers (company, note) VALUES
    ('Insurer One',   ''),
    ('Insurer Two',   'ไม่ระบุปี'),
    ('Insurer Three', 'no payments');

INSERT INTO insurance_payments (insurer_id, year, premium, note) VALUES
    (1, 2024, 2500000, ''),
    (1, 2025, 2500000, ''),
    (1, 2026, 2500000, ''),
    (2, NULL, 1200050, 'satang, no year');

-- 2025 has all 12 months with a December bonus; 2026 has January to March.
INSERT INTO salary_months (year, month, salary, bonus, withholding_tax, pvd, sso)
SELECT 2025, m, 5000000, CASE WHEN m = 12 THEN 10000000 ELSE 0 END, 200000, 300000, 70000
FROM generate_series(1, 12) AS m;

INSERT INTO salary_months (year, month, salary, bonus, withholding_tax, pvd, sso)
SELECT 2026, m, 5500000, 0, 250000, 330000, 80000
FROM generate_series(1, 3) AS m;

INSERT INTO tax_deductions (year, item, amount) VALUES
    (2025, 'ส่วนตัว',        6000000),
    (2025, 'ประกันสังคม',    900000),
    (2025, 'Life insurance', 2500000),
    (2026, 'ส่วนตัว',        6000000);

COMMIT;

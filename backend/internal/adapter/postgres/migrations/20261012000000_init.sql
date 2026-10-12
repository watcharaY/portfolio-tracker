CREATE TABLE users (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username             TEXT        NOT NULL UNIQUE,
    password_hash        TEXT        NOT NULL,
    must_change_password BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip         TEXT        NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_sessions_user    ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE brokers (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          TEXT        NOT NULL UNIQUE,
    current_value BIGINT      NOT NULL DEFAULT 0 CHECK (current_value >= 0),
    note          TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE broker_transactions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    broker_id  BIGINT      NOT NULL REFERENCES brokers(id) ON DELETE CASCADE,
    txn_date   DATE,
    type       TEXT        NOT NULL CHECK (type IN ('deposit', 'withdraw')),
    amount     BIGINT      NOT NULL CHECK (amount > 0),
    note       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_broker_txn_broker ON broker_transactions(broker_id);

CREATE TABLE funds (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         TEXT        NOT NULL UNIQUE,
    category     TEXT        NOT NULL,
    cost         BIGINT      NOT NULL DEFAULT 0 CHECK (cost >= 0),
    market_value BIGINT      NOT NULL DEFAULT 0 CHECK (market_value >= 0),
    note         TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pvd_years (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    year                 INTEGER     NOT NULL UNIQUE,
    self_contribution    BIGINT      NOT NULL DEFAULT 0,
    company_contribution BIGINT      NOT NULL DEFAULT 0,
    self_profit          BIGINT      NOT NULL DEFAULT 0,
    company_profit       BIGINT      NOT NULL DEFAULT 0,
    fund_allocation      TEXT        NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE savings_accounts (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bank        TEXT        NOT NULL,
    balance     BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
    interest_bp INTEGER     NOT NULL DEFAULT 0 CHECK (interest_bp BETWEEN 0 AND 10000),
    note        TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE insurers (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company    TEXT        NOT NULL UNIQUE,
    note       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE insurance_payments (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    insurer_id BIGINT      NOT NULL REFERENCES insurers(id) ON DELETE CASCADE,
    year       INTEGER,
    premium    BIGINT      NOT NULL CHECK (premium > 0),
    note       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ins_pay_insurer ON insurance_payments(insurer_id);

CREATE TABLE salary_months (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    year            INTEGER     NOT NULL,
    month           INTEGER     NOT NULL CHECK (month BETWEEN 1 AND 12),
    salary          BIGINT      NOT NULL DEFAULT 0,
    bonus           BIGINT      NOT NULL DEFAULT 0,
    withholding_tax BIGINT      NOT NULL DEFAULT 0,
    pvd             BIGINT      NOT NULL DEFAULT 0,
    sso             BIGINT      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (year, month)
);

CREATE TABLE tax_deductions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    year       INTEGER     NOT NULL,
    item       TEXT        NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tax_deductions_year ON tax_deductions(year);

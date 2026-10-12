-- dividend_payments: cash an asset paid out (stock or fund dividend, savings
-- interest, insurance payout). Exactly one source column is set; each cascades
-- with its asset like the other child tables.
CREATE TABLE dividend_payments (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    broker_id          BIGINT REFERENCES brokers(id) ON DELETE CASCADE,
    fund_id            BIGINT REFERENCES funds(id) ON DELETE CASCADE,
    savings_account_id BIGINT REFERENCES savings_accounts(id) ON DELETE CASCADE,
    insurer_id         BIGINT REFERENCES insurers(id) ON DELETE CASCADE,
    paid_on            DATE        NOT NULL,
    gross              BIGINT      NOT NULL CHECK (gross > 0),
    withholding_tax    BIGINT      NOT NULL DEFAULT 0 CHECK (withholding_tax >= 0 AND withholding_tax <= gross),
    note               TEXT        NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (num_nonnulls(broker_id, fund_id, savings_account_id, insurer_id) = 1)
);
CREATE INDEX idx_dividend_payments_paid_on ON dividend_payments(paid_on);
CREATE INDEX idx_dividend_payments_broker  ON dividend_payments(broker_id);
CREATE INDEX idx_dividend_payments_fund    ON dividend_payments(fund_id);
CREATE INDEX idx_dividend_payments_savings ON dividend_payments(savings_account_id);
CREATE INDEX idx_dividend_payments_insurer ON dividend_payments(insurer_id);

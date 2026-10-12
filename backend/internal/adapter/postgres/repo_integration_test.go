package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

var (
	setupOnce sync.Once
	sharedDB  *postgres.DB
	setupErr  error
)

// openDB returns a migrated database with every table emptied and identities reset,
// so ids in each test start at 1.
func openDB(t *testing.T) *postgres.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	setupOnce.Do(func() {
		sharedDB, setupErr = postgres.Connect(ctx, url)
		if setupErr == nil {
			setupErr = postgres.Migrate(ctx, sharedDB)
		}
	})
	if setupErr != nil {
		t.Fatalf("setup: %v", setupErr)
	}
	exec(t, `TRUNCATE users, sessions, brokers, broker_transactions, funds, pvd_years, savings_accounts,
	         insurers, insurance_payments, salary_months, tax_deductions RESTART IDENTITY CASCADE`)
	return sharedDB
}

// exec runs raw SQL on its own connection, independent of the adapter under test.
func exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func scalar[T any](t *testing.T, sql string, args ...any) T {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	var v T
	if err := conn.QueryRow(ctx, sql, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return v
}

// must takes a (value, error) call directly, as in must(repo.Get(ctx, id))(t).
func must[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return v
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}

func wantValidation(t *testing.T, err error, field string) {
	t.Helper()
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v (%T), want *domain.ValidationError", err, err)
	}
	if ve.Field != field {
		t.Fatalf("ValidationError.Field = %q, want %q", ve.Field, field)
	}
}

func wantRows(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("rows mismatch\n got: %q\nwant: %q", got, want)
	}
}

func rows[T any](items []T, f func(T) string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = f(it)
	}
	return out
}

// wantTimestamps asserts that Update kept created_at and advanced updated_at.
func wantTimestamps(t *testing.T, beforeCreated, beforeUpdated, afterCreated, afterUpdated time.Time) {
	t.Helper()
	if beforeCreated.IsZero() || beforeUpdated.IsZero() {
		t.Fatalf("timestamps not set: created %v updated %v", beforeCreated, beforeUpdated)
	}
	if !afterCreated.Equal(beforeCreated) {
		t.Fatalf("created_at changed on update: %v -> %v", beforeCreated, afterCreated)
	}
	if !afterUpdated.After(beforeUpdated) {
		t.Fatalf("updated_at did not advance: %v -> %v", beforeUpdated, afterUpdated)
	}
}

func date(y int, m time.Month, d int) *time.Time {
	v := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &v
}

func ptr[T any](v T) *T { return &v }

func fmtDate(d *time.Time) string {
	if d == nil {
		return "nil"
	}
	return d.Format("2006-01-02")
}

func fmtYear(y *int) string {
	if y == nil {
		return "nil"
	}
	return fmt.Sprint(*y)
}

var ctx = context.Background()

func TestMigrateIsIdempotent(t *testing.T) {
	db := openDB(t)
	var brokers usecase.BrokerRepository = postgres.NewBrokerRepo(db)
	must(brokers.Create(ctx, &domain.Broker{Name: "Keep"}))(t)
	before := scalar[int](t, "SELECT COUNT(*)::INT FROM schema_migrations")

	noErr(t, postgres.Migrate(ctx, db))

	if after := scalar[int](t, "SELECT COUNT(*)::INT FROM schema_migrations"); after != before || after != 1 {
		t.Fatalf("schema_migrations rows = %d after second Migrate, want %d (1)", after, before)
	}
	got := must(brokers.List(ctx, domain.NoFilter{}))(t)
	wantRows(t, rows(got, func(b domain.BrokerSummary) string { return b.Name }), []string{"Keep"})
}

func TestNotFound(t *testing.T) {
	db := openDB(t)
	const missing = 999
	cases := map[string]struct {
		get, update, del func() error
	}{
		"broker": {
			func() error { _, err := postgres.NewBrokerRepo(db).Get(ctx, missing); return err },
			func() error { return postgres.NewBrokerRepo(db).Update(ctx, missing, &domain.Broker{Name: "x"}) },
			func() error { return postgres.NewBrokerRepo(db).Delete(ctx, missing) },
		},
		"broker transaction": {
			func() error { _, err := postgres.NewBrokerTransactionRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewBrokerTransactionRepo(db).Update(ctx, missing,
					&domain.BrokerTransaction{BrokerID: 1, Type: domain.TxnDeposit, Amount: 1})
			},
			func() error { return postgres.NewBrokerTransactionRepo(db).Delete(ctx, missing) },
		},
		"fund": {
			func() error { _, err := postgres.NewFundRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewFundRepo(db).Update(ctx, missing, &domain.Fund{Name: "x", Category: "Gold"})
			},
			func() error { return postgres.NewFundRepo(db).Delete(ctx, missing) },
		},
		"pvd": {
			func() error { _, err := postgres.NewPVDRepo(db).Get(ctx, missing); return err },
			func() error { return postgres.NewPVDRepo(db).Update(ctx, missing, &domain.PVDYear{Year: 2025}) },
			func() error { return postgres.NewPVDRepo(db).Delete(ctx, missing) },
		},
		"savings": {
			func() error { _, err := postgres.NewSavingsRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewSavingsRepo(db).Update(ctx, missing, &domain.SavingsAccount{Bank: "x"})
			},
			func() error { return postgres.NewSavingsRepo(db).Delete(ctx, missing) },
		},
		"insurer": {
			func() error { _, err := postgres.NewInsurerRepo(db).Get(ctx, missing); return err },
			func() error { return postgres.NewInsurerRepo(db).Update(ctx, missing, &domain.Insurer{Company: "x"}) },
			func() error { return postgres.NewInsurerRepo(db).Delete(ctx, missing) },
		},
		"insurance payment": {
			func() error { _, err := postgres.NewInsurancePaymentRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewInsurancePaymentRepo(db).Update(ctx, missing,
					&domain.InsurancePayment{InsurerID: 1, Premium: 1})
			},
			func() error { return postgres.NewInsurancePaymentRepo(db).Delete(ctx, missing) },
		},
		"salary": {
			func() error { _, err := postgres.NewSalaryRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewSalaryRepo(db).Update(ctx, missing, &domain.SalaryMonth{Year: 2025, Month: 1})
			},
			func() error { return postgres.NewSalaryRepo(db).Delete(ctx, missing) },
		},
		"tax deduction": {
			func() error { _, err := postgres.NewTaxDeductionRepo(db).Get(ctx, missing); return err },
			func() error {
				return postgres.NewTaxDeductionRepo(db).Update(ctx, missing, &domain.TaxDeduction{Year: 2025, Item: "x"})
			},
			func() error { return postgres.NewTaxDeductionRepo(db).Delete(ctx, missing) },
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			wantErr(t, c.get(), domain.ErrNotFound)
			wantErr(t, c.update(), domain.ErrNotFound)
			wantErr(t, c.del(), domain.ErrNotFound)
		})
	}
}

func TestBrokerRepo(t *testing.T) {
	db := openDB(t)
	var brokers usecase.BrokerRepository = postgres.NewBrokerRepo(db)
	var txns usecase.BrokerTransactionRepository = postgres.NewBrokerTransactionRepo(db)

	zeta := must(brokers.Create(ctx, &domain.Broker{Name: "Zeta", CurrentValue: 5_000, Note: "z"}))(t)
	alpha := must(brokers.Create(ctx, &domain.Broker{Name: "Alpha", CurrentValue: 1_000}))(t)
	if zeta != 1 || alpha != 2 {
		t.Fatalf("ids = %d, %d; want 1, 2", zeta, alpha)
	}
	for _, tx := range []domain.BrokerTransaction{
		{BrokerID: zeta, Type: domain.TxnDeposit, Amount: 100},
		{BrokerID: zeta, Type: domain.TxnDeposit, Amount: 250},
		{BrokerID: zeta, Type: domain.TxnWithdraw, Amount: 40},
	} {
		must(txns.Create(ctx, &tx))(t)
	}

	fmtSummary := func(b domain.BrokerSummary) string {
		return fmt.Sprintf("%d %s value=%d note=%q dep=%d wd=%d", b.ID, b.Name, b.CurrentValue, b.Note, b.Deposits, b.Withdrawals)
	}
	list := must(brokers.List(ctx, domain.NoFilter{}))(t)
	wantRows(t, rows(list, fmtSummary), []string{
		`2 Alpha value=1000 note="" dep=0 wd=0`,
		`1 Zeta value=5000 note="z" dep=350 wd=40`,
	})

	before := must(brokers.Get(ctx, zeta))(t)
	wantRows(t, []string{fmtSummary(before)}, []string{`1 Zeta value=5000 note="z" dep=350 wd=40`})

	noErr(t, brokers.Update(ctx, zeta, &domain.Broker{Name: "Zulu", CurrentValue: 7_000, Note: "renamed"}))
	after := must(brokers.Get(ctx, zeta))(t)
	wantRows(t, []string{fmtSummary(after)}, []string{`1 Zulu value=7000 note="renamed" dep=350 wd=40`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	t.Run("duplicate name is a conflict", func(t *testing.T) {
		_, err := brokers.Create(ctx, &domain.Broker{Name: "Alpha"})
		wantErr(t, err, domain.ErrConflict)
		wantErr(t, brokers.Update(ctx, zeta, &domain.Broker{Name: "Alpha"}), domain.ErrConflict)
	})

	t.Run("check violation names the constraint", func(t *testing.T) {
		_, err := brokers.Create(ctx, &domain.Broker{Name: "Neg", CurrentValue: -1})
		wantValidation(t, err, "brokers_current_value_check")
	})

	t.Run("delete cascades to transactions", func(t *testing.T) {
		noErr(t, brokers.Delete(ctx, zeta))
		wantErr(t, func() error { _, err := brokers.Get(ctx, zeta); return err }(), domain.ErrNotFound)
		left := must(txns.List(ctx, domain.BrokerTransactionFilter{}))(t)
		wantRows(t, rows(left, func(tx domain.BrokerTransaction) string { return fmt.Sprint(tx.ID) }), []string{})
		if n := scalar[int](t, "SELECT COUNT(*)::INT FROM broker_transactions"); n != 0 {
			t.Fatalf("broker_transactions rows = %d after deleting broker, want 0", n)
		}
	})
}

func TestBrokerTransactionRepo(t *testing.T) {
	db := openDB(t)
	var brokers usecase.BrokerRepository = postgres.NewBrokerRepo(db)
	var txns usecase.BrokerTransactionRepository = postgres.NewBrokerTransactionRepo(db)

	a := must(brokers.Create(ctx, &domain.Broker{Name: "A"}))(t)
	b := must(brokers.Create(ctx, &domain.Broker{Name: "B"}))(t)
	for _, tx := range []domain.BrokerTransaction{
		{BrokerID: a, Date: date(2025, 1, 10), Type: domain.TxnDeposit, Amount: 100, Note: "first"},
		{BrokerID: b, Date: nil, Type: domain.TxnDeposit, Amount: 200},
		{BrokerID: a, Date: date(2025, 3, 5), Type: domain.TxnWithdraw, Amount: 30},
		{BrokerID: a, Date: nil, Type: domain.TxnDeposit, Amount: 400},
		{BrokerID: b, Date: date(2025, 3, 5), Type: domain.TxnDeposit, Amount: 500},
	} {
		must(txns.Create(ctx, &tx))(t)
	}

	fmtTxn := func(tx domain.BrokerTransaction) string {
		return fmt.Sprintf("%d %d/%s %s %s %d %q", tx.ID, tx.BrokerID, tx.BrokerName, fmtDate(tx.Date), tx.Type, tx.Amount, tx.Note)
	}
	t.Run("ordered by date desc nulls last then id desc", func(t *testing.T) {
		wantRows(t, rows(must(txns.List(ctx, domain.BrokerTransactionFilter{}))(t), fmtTxn), []string{
			`5 2/B 2025-03-05 deposit 500 ""`,
			`3 1/A 2025-03-05 withdraw 30 ""`,
			`1 1/A 2025-01-10 deposit 100 "first"`,
			`4 1/A nil deposit 400 ""`,
			`2 2/B nil deposit 200 ""`,
		})
	})

	t.Run("filter by broker", func(t *testing.T) {
		wantRows(t, rows(must(txns.List(ctx, domain.BrokerTransactionFilter{BrokerID: &a}))(t), fmtTxn), []string{
			`3 1/A 2025-03-05 withdraw 30 ""`,
			`1 1/A 2025-01-10 deposit 100 "first"`,
			`4 1/A nil deposit 400 ""`,
		})
	})

	t.Run("nullable date round-trips through update", func(t *testing.T) {
		before := must(txns.Get(ctx, 2))(t)
		wantRows(t, []string{fmtTxn(before)}, []string{`2 2/B nil deposit 200 ""`})

		noErr(t, txns.Update(ctx, 2, &domain.BrokerTransaction{BrokerID: a, Date: date(2024, 12, 31), Type: domain.TxnWithdraw, Amount: 20, Note: "moved"}))
		after := must(txns.Get(ctx, 2))(t)
		wantRows(t, []string{fmtTxn(after)}, []string{`2 1/A 2024-12-31 withdraw 20 "moved"`})
		wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

		noErr(t, txns.Update(ctx, 2, &domain.BrokerTransaction{BrokerID: a, Date: nil, Type: domain.TxnWithdraw, Amount: 20}))
		wantRows(t, []string{fmtTxn(must(txns.Get(ctx, 2))(t))}, []string{`2 1/A nil withdraw 20 ""`})
	})

	t.Run("missing broker is a reference error", func(t *testing.T) {
		_, err := txns.Create(ctx, &domain.BrokerTransaction{BrokerID: 999, Type: domain.TxnDeposit, Amount: 1})
		wantErr(t, err, domain.ErrReferenceNotFound)
		err = txns.Update(ctx, 1, &domain.BrokerTransaction{BrokerID: 999, Type: domain.TxnDeposit, Amount: 1})
		wantErr(t, err, domain.ErrReferenceNotFound)
	})

	t.Run("check violations name the constraint", func(t *testing.T) {
		_, err := txns.Create(ctx, &domain.BrokerTransaction{BrokerID: a, Type: domain.TxnDeposit, Amount: 0})
		wantValidation(t, err, "broker_transactions_amount_check")
		_, err = txns.Create(ctx, &domain.BrokerTransaction{BrokerID: a, Type: "buy", Amount: 1})
		wantValidation(t, err, "broker_transactions_type_check")
	})

	t.Run("delete", func(t *testing.T) {
		noErr(t, txns.Delete(ctx, 5))
		wantErr(t, func() error { _, err := txns.Get(ctx, 5); return err }(), domain.ErrNotFound)
		got := must(brokers.Get(ctx, b))(t)
		if got.Deposits != 0 || got.Withdrawals != 0 {
			t.Fatalf("broker B sums = %d/%d after deleting its txns, want 0/0", got.Deposits, got.Withdrawals)
		}
	})
}

func TestFundRepo(t *testing.T) {
	db := openDB(t)
	var funds usecase.FundRepository = postgres.NewFundRepo(db)

	must(funds.Create(ctx, &domain.Fund{Name: "Mid", Category: "Gold", Cost: 100, MarketValue: 120, Note: "n"}))(t)
	must(funds.Create(ctx, &domain.Fund{Name: "Apex", Category: "RMF", Cost: 50, MarketValue: 40}))(t)

	fmtFund := func(f domain.Fund) string {
		return fmt.Sprintf("%d %s %s %d %d %q", f.ID, f.Name, f.Category, f.Cost, f.MarketValue, f.Note)
	}
	wantRows(t, rows(must(funds.List(ctx, domain.NoFilter{}))(t), fmtFund), []string{
		`2 Apex RMF 50 40 ""`,
		`1 Mid Gold 100 120 "n"`,
	})

	before := must(funds.Get(ctx, 1))(t)
	noErr(t, funds.Update(ctx, 1, &domain.Fund{Name: "Mid2", Category: "SSF", Cost: 200, MarketValue: 210, Note: "u"}))
	after := must(funds.Get(ctx, 1))(t)
	wantRows(t, []string{fmtFund(after)}, []string{`1 Mid2 SSF 200 210 "u"`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := funds.Create(ctx, &domain.Fund{Name: "Apex", Category: "Gold"})
	wantErr(t, err, domain.ErrConflict)
	_, err = funds.Create(ctx, &domain.Fund{Name: "Neg", Category: "Gold", Cost: -1})
	wantValidation(t, err, "funds_cost_check")

	noErr(t, funds.Delete(ctx, 2))
	wantRows(t, rows(must(funds.List(ctx, domain.NoFilter{}))(t), fmtFund), []string{`1 Mid2 SSF 200 210 "u"`})
}

func TestPVDRepo(t *testing.T) {
	db := openDB(t)
	var pvd usecase.PVDRepository = postgres.NewPVDRepo(db)

	must(pvd.Create(ctx, &domain.PVDYear{Year: 2025, SelfContribution: 10, CompanyContribution: 5, SelfProfit: -2, CompanyProfit: 1, FundAllocation: "60/40"}))(t)
	must(pvd.Create(ctx, &domain.PVDYear{Year: 2023, SelfContribution: 1}))(t)

	fmtPVD := func(p domain.PVDYear) string {
		return fmt.Sprintf("%d %d %d %d %d %d %q", p.ID, p.Year, p.SelfContribution, p.CompanyContribution, p.SelfProfit, p.CompanyProfit, p.FundAllocation)
	}
	wantRows(t, rows(must(pvd.List(ctx, domain.NoFilter{}))(t), fmtPVD), []string{
		`2 2023 1 0 0 0 ""`,
		`1 2025 10 5 -2 1 "60/40"`,
	})

	before := must(pvd.Get(ctx, 2))(t)
	noErr(t, pvd.Update(ctx, 2, &domain.PVDYear{Year: 2024, SelfContribution: 7, CompanyContribution: 3, SelfProfit: 2, CompanyProfit: -1, FundAllocation: "x"}))
	after := must(pvd.Get(ctx, 2))(t)
	wantRows(t, []string{fmtPVD(after)}, []string{`2 2024 7 3 2 -1 "x"`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := pvd.Create(ctx, &domain.PVDYear{Year: 2025})
	wantErr(t, err, domain.ErrConflict)

	noErr(t, pvd.Delete(ctx, 1))
	wantRows(t, rows(must(pvd.List(ctx, domain.NoFilter{}))(t), fmtPVD), []string{`2 2024 7 3 2 -1 "x"`})
}

func TestSavingsRepo(t *testing.T) {
	db := openDB(t)
	var savings usecase.SavingsRepository = postgres.NewSavingsRepo(db)

	must(savings.Create(ctx, &domain.SavingsAccount{Bank: "KBank", Balance: 300, InterestRate: 150, Note: "b"}))(t)
	must(savings.Create(ctx, &domain.SavingsAccount{Bank: "Bangkok", Balance: 100, InterestRate: 50}))(t)
	must(savings.Create(ctx, &domain.SavingsAccount{Bank: "KBank", Balance: 200, InterestRate: 0}))(t)

	fmtSavings := func(s domain.SavingsAccount) string {
		return fmt.Sprintf("%d %s %d %d %q", s.ID, s.Bank, s.Balance, s.InterestRate, s.Note)
	}
	wantRows(t, rows(must(savings.List(ctx, domain.NoFilter{}))(t), fmtSavings), []string{
		`2 Bangkok 100 50 ""`,
		`1 KBank 300 150 "b"`,
		`3 KBank 200 0 ""`,
	})

	before := must(savings.Get(ctx, 3))(t)
	noErr(t, savings.Update(ctx, 3, &domain.SavingsAccount{Bank: "SCB", Balance: 900, InterestRate: 10_000, Note: "max"}))
	after := must(savings.Get(ctx, 3))(t)
	wantRows(t, []string{fmtSavings(after)}, []string{`3 SCB 900 10000 "max"`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := savings.Create(ctx, &domain.SavingsAccount{Bank: "X", InterestRate: 10_001})
	wantValidation(t, err, "savings_accounts_interest_bp_check")
	_, err = savings.Create(ctx, &domain.SavingsAccount{Bank: "X", Balance: -1})
	wantValidation(t, err, "savings_accounts_balance_check")

	noErr(t, savings.Delete(ctx, 1))
	wantRows(t, rows(must(savings.List(ctx, domain.NoFilter{}))(t), fmtSavings), []string{
		`2 Bangkok 100 50 ""`,
		`3 SCB 900 10000 "max"`,
	})
}

func TestInsurerRepo(t *testing.T) {
	db := openDB(t)
	var insurers usecase.InsurerRepository = postgres.NewInsurerRepo(db)
	var payments usecase.InsurancePaymentRepository = postgres.NewInsurancePaymentRepo(db)

	muang := must(insurers.Create(ctx, &domain.Insurer{Company: "Muang Thai", Note: "life"}))(t)
	must(insurers.Create(ctx, &domain.Insurer{Company: "AIA"}))(t)
	must(payments.Create(ctx, &domain.InsurancePayment{InsurerID: muang, Year: ptr(2024), Premium: 1_000}))(t)
	must(payments.Create(ctx, &domain.InsurancePayment{InsurerID: muang, Year: ptr(2025), Premium: 1_500}))(t)

	fmtInsurer := func(i domain.InsurerSummary) string {
		return fmt.Sprintf("%d %s %q payments=%d total=%d", i.ID, i.Company, i.Note, i.Payments, i.TotalPaid)
	}
	wantRows(t, rows(must(insurers.List(ctx, domain.NoFilter{}))(t), fmtInsurer), []string{
		`2 AIA "" payments=0 total=0`,
		`1 Muang Thai "life" payments=2 total=2500`,
	})

	before := must(insurers.Get(ctx, muang))(t)
	noErr(t, insurers.Update(ctx, muang, &domain.Insurer{Company: "MTL", Note: "renamed"}))
	after := must(insurers.Get(ctx, muang))(t)
	wantRows(t, []string{fmtInsurer(after)}, []string{`1 MTL "renamed" payments=2 total=2500`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := insurers.Create(ctx, &domain.Insurer{Company: "AIA"})
	wantErr(t, err, domain.ErrConflict)

	noErr(t, insurers.Delete(ctx, muang))
	wantRows(t, rows(must(insurers.List(ctx, domain.NoFilter{}))(t), fmtInsurer), []string{`2 AIA "" payments=0 total=0`})
	if n := scalar[int](t, "SELECT COUNT(*)::INT FROM insurance_payments"); n != 0 {
		t.Fatalf("insurance_payments rows = %d after deleting insurer, want 0", n)
	}
}

func TestInsurancePaymentRepo(t *testing.T) {
	db := openDB(t)
	var insurers usecase.InsurerRepository = postgres.NewInsurerRepo(db)
	var payments usecase.InsurancePaymentRepository = postgres.NewInsurancePaymentRepo(db)

	zurich := must(insurers.Create(ctx, &domain.Insurer{Company: "Zurich"}))(t)
	aia := must(insurers.Create(ctx, &domain.Insurer{Company: "AIA"}))(t)
	for _, p := range []domain.InsurancePayment{
		{InsurerID: zurich, Year: ptr(2025), Premium: 100},            // 1
		{InsurerID: aia, Year: ptr(2024), Premium: 200},               // 2
		{InsurerID: zurich, Year: nil, Premium: 300, Note: "undated"}, // 3
		{InsurerID: zurich, Year: ptr(2024), Premium: 400},            // 4
		{InsurerID: aia, Year: ptr(2024), Premium: 500},               // 5
		{InsurerID: zurich, Year: ptr(2025), Premium: 600},            // 6
	} {
		must(payments.Create(ctx, &p))(t)
	}

	fmtPayment := func(p domain.InsurancePayment) string {
		return fmt.Sprintf("%s #%d id=%d ins=%d year=%s %d %q", p.Company, p.PaymentNo, p.ID, p.InsurerID, fmtYear(p.Year), p.Premium, p.Note)
	}
	t.Run("ordered by company then payment number, numbered by year nulls first then id", func(t *testing.T) {
		wantRows(t, rows(must(payments.List(ctx, domain.InsurancePaymentFilter{}))(t), fmtPayment), []string{
			`AIA #1 id=2 ins=2 year=2024 200 ""`,
			`AIA #2 id=5 ins=2 year=2024 500 ""`,
			`Zurich #1 id=3 ins=1 year=nil 300 "undated"`,
			`Zurich #2 id=4 ins=1 year=2024 400 ""`,
			`Zurich #3 id=1 ins=1 year=2025 100 ""`,
			`Zurich #4 id=6 ins=1 year=2025 600 ""`,
		})
	})

	t.Run("filter by insurer", func(t *testing.T) {
		wantRows(t, rows(must(payments.List(ctx, domain.InsurancePaymentFilter{InsurerID: &aia}))(t), fmtPayment), []string{
			`AIA #1 id=2 ins=2 year=2024 200 ""`,
			`AIA #2 id=5 ins=2 year=2024 500 ""`,
		})
	})

	t.Run("filter by year keeps the unfiltered numbers", func(t *testing.T) {
		wantRows(t, rows(must(payments.List(ctx, domain.InsurancePaymentFilter{Year: ptr(2025)}))(t), fmtPayment), []string{
			`Zurich #3 id=1 ins=1 year=2025 100 ""`,
			`Zurich #4 id=6 ins=1 year=2025 600 ""`,
		})
		wantRows(t, rows(must(payments.List(ctx, domain.InsurancePaymentFilter{InsurerID: &zurich, Year: ptr(2024)}))(t), fmtPayment), []string{
			`Zurich #2 id=4 ins=1 year=2024 400 ""`,
		})
	})

	t.Run("get carries the payment number", func(t *testing.T) {
		wantRows(t, []string{fmtPayment(must(payments.Get(ctx, 6))(t))}, []string{`Zurich #4 id=6 ins=1 year=2025 600 ""`})
	})

	t.Run("nullable year round-trips and renumbers", func(t *testing.T) {
		before := must(payments.Get(ctx, 1))(t)
		noErr(t, payments.Update(ctx, 1, &domain.InsurancePayment{InsurerID: zurich, Year: nil, Premium: 150, Note: "cleared"}))
		after := must(payments.Get(ctx, 1))(t)
		wantRows(t, []string{fmtPayment(after)}, []string{`Zurich #1 id=1 ins=1 year=nil 150 "cleared"`})
		wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)
		wantRows(t, []string{fmtPayment(must(payments.Get(ctx, 3))(t))}, []string{`Zurich #2 id=3 ins=1 year=nil 300 "undated"`})

		noErr(t, payments.Update(ctx, 1, &domain.InsurancePayment{InsurerID: aia, Year: ptr(2026), Premium: 150}))
		wantRows(t, []string{fmtPayment(must(payments.Get(ctx, 1))(t))}, []string{`AIA #3 id=1 ins=2 year=2026 150 ""`})
	})

	t.Run("missing insurer is a reference error", func(t *testing.T) {
		_, err := payments.Create(ctx, &domain.InsurancePayment{InsurerID: 999, Premium: 1})
		wantErr(t, err, domain.ErrReferenceNotFound)
		wantErr(t, payments.Update(ctx, 2, &domain.InsurancePayment{InsurerID: 999, Premium: 1}), domain.ErrReferenceNotFound)
	})

	t.Run("check violation names the constraint", func(t *testing.T) {
		_, err := payments.Create(ctx, &domain.InsurancePayment{InsurerID: aia, Premium: 0})
		wantValidation(t, err, "insurance_payments_premium_check")
	})

	t.Run("delete renumbers the rest", func(t *testing.T) {
		noErr(t, payments.Delete(ctx, 2))
		wantRows(t, rows(must(payments.List(ctx, domain.InsurancePaymentFilter{InsurerID: &aia}))(t), fmtPayment), []string{
			`AIA #1 id=5 ins=2 year=2024 500 ""`,
			`AIA #2 id=1 ins=2 year=2026 150 ""`,
		})
	})
}

func TestSalaryRepo(t *testing.T) {
	db := openDB(t)
	var salary usecase.SalaryRepository = postgres.NewSalaryRepo(db)

	for _, s := range []domain.SalaryMonth{
		{Year: 2025, Month: 2, Salary: 200, Bonus: 20, WithholdingTax: 2, PVD: 3, SSO: 4},
		{Year: 2024, Month: 12, Salary: 100},
		{Year: 2025, Month: 1, Salary: 150},
	} {
		must(salary.Create(ctx, &s))(t)
	}

	fmtSalary := func(s domain.SalaryMonth) string {
		return fmt.Sprintf("%d %d-%02d %d %d %d %d %d", s.ID, s.Year, s.Month, s.Salary, s.Bonus, s.WithholdingTax, s.PVD, s.SSO)
	}
	wantRows(t, rows(must(salary.List(ctx, domain.YearFilter{}))(t), fmtSalary), []string{
		`2 2024-12 100 0 0 0 0`,
		`3 2025-01 150 0 0 0 0`,
		`1 2025-02 200 20 2 3 4`,
	})
	wantRows(t, rows(must(salary.List(ctx, domain.YearFilter{Year: ptr(2025)}))(t), fmtSalary), []string{
		`3 2025-01 150 0 0 0 0`,
		`1 2025-02 200 20 2 3 4`,
	})

	before := must(salary.Get(ctx, 2))(t)
	noErr(t, salary.Update(ctx, 2, &domain.SalaryMonth{Year: 2024, Month: 11, Salary: 110, Bonus: 1, WithholdingTax: 2, PVD: 3, SSO: 4}))
	after := must(salary.Get(ctx, 2))(t)
	wantRows(t, []string{fmtSalary(after)}, []string{`2 2024-11 110 1 2 3 4`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := salary.Create(ctx, &domain.SalaryMonth{Year: 2025, Month: 1})
	wantErr(t, err, domain.ErrConflict)
	_, err = salary.Create(ctx, &domain.SalaryMonth{Year: 2025, Month: 13})
	wantValidation(t, err, "salary_months_month_check")

	noErr(t, salary.Delete(ctx, 3))
	wantRows(t, rows(must(salary.List(ctx, domain.YearFilter{Year: ptr(2025)}))(t), fmtSalary), []string{`1 2025-02 200 20 2 3 4`})
}

func TestTaxDeductionRepo(t *testing.T) {
	db := openDB(t)
	var deductions usecase.TaxDeductionRepository = postgres.NewTaxDeductionRepo(db)

	for _, d := range []domain.TaxDeduction{
		{Year: 2025, Item: "RMF", Amount: 300},
		{Year: 2024, Item: "SSF", Amount: 100},
		{Year: 2025, Item: "Life insurance", Amount: 200},
	} {
		must(deductions.Create(ctx, &d))(t)
	}

	fmtDeduction := func(d domain.TaxDeduction) string {
		return fmt.Sprintf("%d %d %s %d", d.ID, d.Year, d.Item, d.Amount)
	}
	wantRows(t, rows(must(deductions.List(ctx, domain.YearFilter{}))(t), fmtDeduction), []string{
		`2 2024 SSF 100`,
		`1 2025 RMF 300`,
		`3 2025 Life insurance 200`,
	})
	wantRows(t, rows(must(deductions.List(ctx, domain.YearFilter{Year: ptr(2025)}))(t), fmtDeduction), []string{
		`1 2025 RMF 300`,
		`3 2025 Life insurance 200`,
	})

	before := must(deductions.Get(ctx, 1))(t)
	noErr(t, deductions.Update(ctx, 1, &domain.TaxDeduction{Year: 2023, Item: "ThaiESG", Amount: 50}))
	after := must(deductions.Get(ctx, 1))(t)
	wantRows(t, []string{fmtDeduction(after)}, []string{`1 2023 ThaiESG 50`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	noErr(t, deductions.Delete(ctx, 2))
	wantRows(t, rows(must(deductions.List(ctx, domain.YearFilter{}))(t), fmtDeduction), []string{
		`1 2023 ThaiESG 50`,
		`3 2025 Life insurance 200`,
	})
}

func TestUserRepo(t *testing.T) {
	db := openDB(t)
	var users usecase.UserRepository = postgres.NewUserRepo(db)

	if n := must(users.Count(ctx))(t); n != 0 {
		t.Fatalf("Count on empty table = %d, want 0", n)
	}
	id := must(users.Create(ctx, domain.User{Username: "admin", PasswordHash: "h1", MustChangePassword: true}))(t)
	must(users.Create(ctx, domain.User{Username: "bob", PasswordHash: "h2"}))(t)
	if n := must(users.Count(ctx))(t); n != 2 {
		t.Fatalf("Count = %d, want 2", n)
	}

	fmtUser := func(u domain.User) string {
		return fmt.Sprintf("%d %s %s must=%t", u.ID, u.Username, u.PasswordHash, u.MustChangePassword)
	}
	before := must(users.GetByUsername(ctx, "admin"))(t)
	wantRows(t, []string{fmtUser(before)}, []string{`1 admin h1 must=true`})
	wantRows(t, []string{fmtUser(must(users.GetByID(ctx, 2))(t))}, []string{`2 bob h2 must=false`})

	noErr(t, users.UpdatePassword(ctx, id, "h3", false))
	after := must(users.GetByID(ctx, id))(t)
	wantRows(t, []string{fmtUser(after)}, []string{`1 admin h3 must=false`})
	wantTimestamps(t, before.CreatedAt, before.UpdatedAt, after.CreatedAt, after.UpdatedAt)

	_, err := users.Create(ctx, domain.User{Username: "admin", PasswordHash: "x"})
	wantErr(t, err, domain.ErrConflict)
	_, err = users.GetByUsername(ctx, "nobody")
	wantErr(t, err, domain.ErrNotFound)
	_, err = users.GetByID(ctx, 999)
	wantErr(t, err, domain.ErrNotFound)
	wantErr(t, users.UpdatePassword(ctx, 999, "h", false), domain.ErrNotFound)
}

func TestSessionRepo(t *testing.T) {
	db := openDB(t)
	var users usecase.UserRepository = postgres.NewUserRepo(db)
	var sessions usecase.SessionRepository = postgres.NewSessionRepo(db)

	alice := must(users.Create(ctx, domain.User{Username: "alice", PasswordHash: "h"}))(t)
	bob := must(users.Create(ctx, domain.User{Username: "bob", PasswordHash: "h"}))(t)

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, s := range []domain.Session{
		{TokenHash: "a1", UserID: alice, IP: "10.0.0.1", UserAgent: "curl", CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)},
		{TokenHash: "a2", UserID: alice, CreatedAt: t0, ExpiresAt: t0.Add(2 * time.Hour)},
		{TokenHash: "a3", UserID: alice, CreatedAt: t0, ExpiresAt: t0.Add(3 * time.Hour)},
		{TokenHash: "b1", UserID: bob, CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)},
	} {
		noErr(t, sessions.Create(ctx, s))
	}

	fmtSession := func(s domain.Session) string {
		return fmt.Sprintf("%s user=%d ip=%q ua=%q created=%s expires=%s", s.TokenHash, s.UserID, s.IP, s.UserAgent,
			s.CreatedAt.UTC().Format(time.RFC3339), s.ExpiresAt.UTC().Format(time.RFC3339))
	}
	valid := func(token string, now time.Time) error { _, err := sessions.GetValid(ctx, token, now); return err }

	t.Run("get valid respects expiry", func(t *testing.T) {
		got := must(sessions.GetValid(ctx, "a1", t0.Add(59*time.Minute)))(t)
		wantRows(t, []string{fmtSession(got)}, []string{
			`a1 user=1 ip="10.0.0.1" ua="curl" created=2026-01-01T12:00:00Z expires=2026-01-01T13:00:00Z`,
		})
		wantErr(t, valid("a1", t0.Add(time.Hour)), domain.ErrNotFound)
		wantErr(t, valid("nope", t0), domain.ErrNotFound)
	})

	t.Run("duplicate token is a conflict and missing user a reference error", func(t *testing.T) {
		wantErr(t, sessions.Create(ctx, domain.Session{TokenHash: "a1", UserID: alice, ExpiresAt: t0}), domain.ErrConflict)
		wantErr(t, sessions.Create(ctx, domain.Session{TokenHash: "z", UserID: 999, ExpiresAt: t0}), domain.ErrReferenceNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		noErr(t, sessions.Delete(ctx, "a3"))
		wantErr(t, valid("a3", t0), domain.ErrNotFound)
		noErr(t, sessions.Delete(ctx, "a3"))
	})

	t.Run("delete others keeps one session and other users", func(t *testing.T) {
		noErr(t, sessions.Create(ctx, domain.Session{TokenHash: "a3", UserID: alice, CreatedAt: t0, ExpiresAt: t0.Add(3 * time.Hour)}))
		noErr(t, sessions.DeleteOthers(ctx, alice, "a2"))
		wantErr(t, valid("a1", t0), domain.ErrNotFound)
		wantErr(t, valid("a3", t0), domain.ErrNotFound)
		noErr(t, valid("a2", t0))
		noErr(t, valid("b1", t0))
	})

	t.Run("delete expired removes sessions at or before now", func(t *testing.T) {
		noErr(t, sessions.DeleteExpired(ctx, t0.Add(time.Hour)))
		if n := scalar[int](t, "SELECT COUNT(*)::INT FROM sessions"); n != 1 {
			t.Fatalf("sessions left = %d, want 1", n)
		}
		noErr(t, valid("a2", t0))
		wantErr(t, valid("b1", t0), domain.ErrNotFound)
	})
}

func TestWithinTx(t *testing.T) {
	db := openDB(t)
	var tx usecase.TxManager = db
	var brokers usecase.BrokerRepository = postgres.NewBrokerRepo(db)
	names := func() []string {
		return rows(must(brokers.List(ctx, domain.NoFilter{}))(t), func(b domain.BrokerSummary) string { return b.Name })
	}

	t.Run("returned error rolls back", func(t *testing.T) {
		boom := errors.New("boom")
		err := tx.WithinTx(ctx, func(ctx context.Context) error {
			must(brokers.Create(ctx, &domain.Broker{Name: "Rolled"}))(t)
			if got := must(brokers.List(ctx, domain.NoFilter{}))(t); len(got) != 1 {
				t.Errorf("write not visible inside tx: %d rows", len(got))
			}
			return boom
		})
		wantErr(t, err, boom)
		wantRows(t, names(), []string{})
	})

	t.Run("nested tx reuses the outer one", func(t *testing.T) {
		boom := errors.New("boom")
		err := tx.WithinTx(ctx, func(ctx context.Context) error {
			noErr(t, tx.WithinTx(ctx, func(ctx context.Context) error {
				_, err := brokers.Create(ctx, &domain.Broker{Name: "Inner"})
				return err
			}))
			return boom
		})
		wantErr(t, err, boom)
		wantRows(t, names(), []string{})
	})

	t.Run("nil commits", func(t *testing.T) {
		noErr(t, tx.WithinTx(ctx, func(ctx context.Context) error {
			_, err := brokers.Create(ctx, &domain.Broker{Name: "Kept"})
			return err
		}))
		wantRows(t, names(), []string{"Kept"})
	})
}

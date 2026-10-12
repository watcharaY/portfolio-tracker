package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func approx(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBrokerSummary_NetCapitalAndPL(t *testing.T) {
	b := BrokerSummary{
		CurrentValue: 120_000_000,
		Deposits:     100_000_000 + 20_000_000,
		Withdrawals:  10_000_000,
	}
	if got, want := b.NetCapital(), Money(110_000_000); got != want {
		t.Fatalf("net capital = %d, want %d", got, want)
	}
	if got, want := b.UnrealisedPL(), Money(10_000_000); got != want {
		t.Fatalf("P/L = %d, want %d", got, want)
	}
	approx(t, b.PLPercent(), 10_000_000.0/110_000_000.0)
}

func TestBrokerSummary_NoCapitalHasNoPercent(t *testing.T) {
	b := BrokerSummary{CurrentValue: 100, Withdrawals: 50}
	if b.PLPercent() != nil {
		t.Fatal("P/L % must be empty when net capital <= 0")
	}
}

func TestSavingsInterest(t *testing.T) {
	cases := []struct {
		balance Money
		rate    BasisPoints
		want    Money
	}{
		{1_000_000, 250, 25_000},
		{2_000_000, 220, 44_000},
		{0, 300, 0},
	}
	for _, c := range cases {
		s := SavingsAccount{Balance: c.balance, InterestRate: c.rate}
		if got := s.InterestPerYear(); got != c.want {
			t.Errorf("%d @ %dbp = %d, want %d", c.balance, c.rate, got, c.want)
		}
	}
}

func TestPVDYear(t *testing.T) {
	p := PVDYear{Year: 2026, SelfContribution: 3_600_000, CompanyContribution: 1_800_000, SelfProfit: -120_000, CompanyProfit: -60_000}
	if p.TotalContribution() != 5_400_000 || p.TotalProfit() != -180_000 || p.Value() != 5_220_000 {
		t.Fatalf("unexpected totals: %d %d %d", p.TotalContribution(), p.TotalProfit(), p.Value())
	}
	approx(t, p.ReturnPercent(), -180_000.0/5_400_000.0)
}

func TestSummarize(t *testing.T) {
	s := Summarize(Holdings{
		Brokers:   []BrokerSummary{{CurrentValue: 1_100, Deposits: 1_000}},
		Funds:     []Fund{{Cost: 400, MarketValue: 500}},
		PVD:       []PVDYear{{SelfContribution: 200, SelfProfit: 50}},
		Savings:   []SavingsAccount{{Balance: 10_000, InterestRate: 250}},
		Insurance: []InsurerSummary{{TotalPaid: 300}},
	})

	byClass := map[AssetClass]AssetClassSummary{}
	for _, c := range s.Classes {
		byClass[c.Class] = c
	}
	if c := byClass[AssetStocks]; c.Value != 1_100 || c.Cost != 1_000 || c.PL != 100 {
		t.Errorf("stocks = %+v", c)
	}
	if c := byClass[AssetSavings]; c.Value != c.Cost || c.PL != 0 {
		t.Errorf("savings must have no P/L: %+v", c)
	}
	if c := byClass[AssetInsurance]; c.Value != 300 || c.Cost != 300 {
		t.Errorf("insurance = %+v", c)
	}
	if s.Total.Value != 1_100+500+250+10_000+300 {
		t.Errorf("total value = %d", s.Total.Value)
	}
	if s.EstSavingsInterestPerYear != 250 {
		t.Errorf("interest = %d", s.EstSavingsInterestPerYear)
	}
	var sum float64
	for _, c := range s.Classes {
		sum += *c.Allocation
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Errorf("allocations sum to %v, want 1", sum)
	}
}

func TestSummarize_EmptyPortfolio(t *testing.T) {
	s := Summarize(Holdings{})
	if s.Total.Value != 0 || s.Total.PLPercent != nil || s.Total.Allocation != nil {
		t.Fatalf("empty portfolio: %+v", s.Total)
	}
	if len(s.Classes) != 5 {
		t.Fatalf("want all 5 classes, got %d", len(s.Classes))
	}
}

func TestValidation(t *testing.T) {
	year := 1800
	cases := map[string]interface{ Validate() error }{
		"broker without name":       &Broker{Name: "  "},
		"negative broker value":     &Broker{Name: "A", CurrentValue: -1},
		"txn bad type":              &BrokerTransaction{BrokerID: 1, Type: "buy", Amount: 1},
		"txn zero amount":           &BrokerTransaction{BrokerID: 1, Type: TxnDeposit},
		"fund unknown category":     &Fund{Name: "F", Category: "Crypto"},
		"savings rate over 100%":    &SavingsAccount{Bank: "B", InterestRate: 10_001},
		"payment year out of range": &InsurancePayment{InsurerID: 1, Premium: 1, Year: &year},
		"salary month 13":           &SalaryMonth{Year: 2026, Month: 13},
		"deduction without item":    &TaxDeduction{Year: 2026},
	}
	for name, e := range cases {
		err := e.Validate()
		if _, ok := errors.AsType[*ValidationError](err); !ok {
			t.Errorf("%s: want ValidationError, got %v", name, err)
		}
	}

	ok := &Fund{Name: "  Fund A ", Category: "Fixed Income", Cost: 1_000, MarketValue: 1_100}
	if err := ok.Validate(); err != nil || ok.Name != "Fund A" {
		t.Fatalf("valid fund rejected or not trimmed: %v %q", err, ok.Name)
	}
}

func TestValidateNewPassword(t *testing.T) {
	rejected := map[string]struct{ password, reason string }{
		"too short":                 {"Ab1-short", "at least 12"},
		"too long":                  {"Aa1-" + strings.Repeat("x", 61), "at most 64 characters"},
		"over 72 bytes in Thai":     {"Aa1-" + strings.Repeat("ก", 30), "at most 72 bytes"},
		"no uppercase":              {"tiger-mango-42", "uppercase"},
		"no lowercase":              {"TIGER-MANGO-42", "lowercase"},
		"no digit":                  {"Tiger-Mango-Rain", "digit"},
		"no symbol":                 {"TigerMango42Rain", "symbol"},
		"Thai is not a case letter": {"เสือมะม่วง-42-ฝน", "lowercase"},
		"repeat of 4":               {"Tiger-Mango-4444", "repeat"},
		"digit sequence":            {"Tiger-Mango-1234", "sequence"},
		"letter sequence":           {"Wxyz-Mango-42-Rain", "sequence"},
		"keyboard row":              {"Qwer-Mango-42-Rain", "sequence"},
		"descending":                {"Tiger-Mango-9876", "sequence"},
		"username":                  {"Somchai-Mango-42", "username"},
		"admin":                     {"Admin@2026-Mango", "username, admin"},
		"portfolio":                 {"My-Portfolio-42!", "username, admin"},
		"leetspeak common":          {"P@ssw0rd2026!", "too common"},
		"common with suffix":        {"Football2026!", "too common"},
		"leetspeak 1 as l":          {"B@s3ba1l-2026!", "too common"},
	}
	for name, c := range rejected {
		err := ValidateNewPassword("somchai", c.password)
		if ve, ok := errors.AsType[*ValidationError](err); !ok || ve.Field != "newPassword" || !strings.Contains(ve.Error(), c.reason) {
			t.Errorf("%s: %q gave %v, want a newPassword error containing %q", name, c.password, err, c.reason)
		}
	}

	for _, ok := range []string{
		"Tiger-Mango-42-Rain",
		"E2e-Tiger-Mango-42",
		"ปลาทู 3 ตัว Kin Rice",
		"Correct Horse 9 Staple",
	} {
		if err := ValidateNewPassword("somchai", ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
}

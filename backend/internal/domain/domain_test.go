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

func TestBrokerSummary_MatchesSpreadsheet(t *testing.T) {
	b := BrokerSummary{
		Broker:      Broker{CurrentValue: 63_791_726},
		Deposits:    2_000 + 5_000_000 + 25_000_000 + 40_000_000 + 7_454_857 + 10_000_000,
		Withdrawals: 30_002_000,
	}
	if got, want := b.NetCapital(), Money(57_454_857); got != want {
		t.Fatalf("net capital = %d, want %d", got, want)
	}
	if got, want := b.UnrealisedPL(), Money(6_336_869); got != want {
		t.Fatalf("P/L = %d, want %d", got, want)
	}
	approx(t, b.PLPercent(), 6_336_869.0/57_454_857.0)
}

func TestBrokerSummary_NoCapitalHasNoPercent(t *testing.T) {
	b := BrokerSummary{Broker: Broker{CurrentValue: 100}, Withdrawals: 50}
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
	p := PVDYear{Year: 2025, SelfContribution: 555_000, CompanyContribution: 222_000, SelfProfit: -2_241, CompanyProfit: -896}
	if p.TotalContribution() != 777_000 || p.TotalProfit() != -3_137 || p.Value() != 773_863 {
		t.Fatalf("unexpected totals: %d %d %d", p.TotalContribution(), p.TotalProfit(), p.Value())
	}
	approx(t, p.ReturnPercent(), -3_137.0/777_000.0)
}

func TestSummarize(t *testing.T) {
	s := Summarize(Holdings{
		Brokers:   []BrokerSummary{{Broker: Broker{CurrentValue: 1_100}, Deposits: 1_000}},
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
		var ve *ValidationError
		if err := e.Validate(); !errors.As(err, &ve) {
			t.Errorf("%s: want ValidationError, got %v", name, err)
		}
	}

	ok := &Fund{Name: "  Fund A ", Category: "Fixed Income", Cost: 1_000, MarketValue: 1_100}
	if err := ok.Validate(); err != nil || ok.Name != "Fund A" {
		t.Fatalf("valid fund rejected or not trimmed: %v %q", err, ok.Name)
	}
}

func TestValidateNewPassword(t *testing.T) {
	for _, pw := range []string{"short", "admin", "Admin123", "ADMIN123", strings.Repeat("x", MaxPasswordBytes+1)} {
		if ValidateNewPassword("admin123", pw) == nil {
			t.Errorf("%q should be rejected", pw)
		}
	}
	if err := ValidateNewPassword("admin", "correct horse"); err != nil {
		t.Errorf("good password rejected: %v", err)
	}
}

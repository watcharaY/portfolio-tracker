package domain

import (
	"slices"
	"strings"
	"time"
)

var FundCategories = []string{"Equity", "Fixed Income", "Money Market", "Mixed", "Gold", "SSF", "RMF", "ThaiESG", "Other"}

type Fund struct {
	ID          int64
	Name        string
	Category    string
	Cost        Money
	MarketValue Money
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (f *Fund) Validate() error {
	f.Name = strings.TrimSpace(f.Name)
	switch {
	case f.Name == "":
		return invalid("name", "is required")
	case !slices.Contains(FundCategories, f.Category):
		return invalid("category", "must be one of "+strings.Join(FundCategories, ", "))
	case f.Cost < 0 || f.MarketValue < 0:
		return invalid("cost", "must not be negative")
	}
	return nil
}

func (f Fund) UnrealisedPL() Money { return f.MarketValue - f.Cost }
func (f Fund) PLPercent() *float64 { return Ratio(f.UnrealisedPL(), f.Cost) }

type PVDYear struct {
	ID                  int64
	Year                int
	SelfContribution    Money
	CompanyContribution Money
	SelfProfit          Money
	CompanyProfit       Money
	FundAllocation      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (p *PVDYear) Validate() error {
	switch {
	case p.Year < 1900 || p.Year > 3000:
		return invalid("year", "is out of range")
	case p.SelfContribution < 0 || p.CompanyContribution < 0:
		return invalid("self_contribution", "must not be negative")
	}
	return nil
}

func (p PVDYear) TotalContribution() Money { return p.SelfContribution + p.CompanyContribution }
func (p PVDYear) TotalProfit() Money       { return p.SelfProfit + p.CompanyProfit }
func (p PVDYear) Value() Money             { return p.TotalContribution() + p.TotalProfit() }
func (p PVDYear) ReturnPercent() *float64  { return Ratio(p.TotalProfit(), p.TotalContribution()) }

type SavingsAccount struct {
	ID           int64
	Bank         string
	Balance      Money
	InterestRate BasisPoints
	Note         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *SavingsAccount) Validate() error {
	s.Bank = strings.TrimSpace(s.Bank)
	switch {
	case s.Bank == "":
		return invalid("bank", "is required")
	case s.Balance < 0:
		return invalid("balance", "must not be negative")
	case s.InterestRate < 0 || s.InterestRate > 10_000:
		return invalid("interest_bp", "must be between 0 and 10000")
	}
	return nil
}

func (s SavingsAccount) InterestPerYear() Money { return s.InterestRate.Apply(s.Balance) }

type Insurer struct {
	ID        int64
	Company   string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (i *Insurer) Validate() error {
	i.Company = strings.TrimSpace(i.Company)
	if i.Company == "" {
		return invalid("company", "is required")
	}
	return nil
}

type InsurerSummary struct {
	Insurer
	Payments  int
	TotalPaid Money
}

type InsurancePayment struct {
	ID        int64
	InsurerID int64
	Company   string
	PaymentNo int
	Year      *int
	Premium   Money
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *InsurancePayment) Validate() error {
	switch {
	case p.InsurerID <= 0:
		return invalid("insurer_id", "is required")
	case p.Premium <= 0:
		return invalid("premium", "must be greater than 0")
	case p.Year != nil && (*p.Year < 1900 || *p.Year > 3000):
		return invalid("year", "is out of range")
	}
	return nil
}

type InsurancePaymentFilter struct {
	InsurerID *int64
	Year      *int
}

type SalaryMonth struct {
	ID             int64
	Year           int
	Month          int
	Salary         Money
	Bonus          Money
	WithholdingTax Money
	PVD            Money
	SSO            Money
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *SalaryMonth) Validate() error {
	switch {
	case s.Year < 1900 || s.Year > 3000:
		return invalid("year", "is out of range")
	case s.Month < 1 || s.Month > 12:
		return invalid("month", "must be between 1 and 12")
	case s.Salary < 0 || s.Bonus < 0 || s.WithholdingTax < 0 || s.PVD < 0 || s.SSO < 0:
		return invalid("salary", "amounts must not be negative")
	}
	return nil
}

type TaxDeduction struct {
	ID        int64
	Year      int
	Item      string
	Amount    Money
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (d *TaxDeduction) Validate() error {
	d.Item = strings.TrimSpace(d.Item)
	switch {
	case d.Year < 1900 || d.Year > 3000:
		return invalid("year", "is out of range")
	case d.Item == "":
		return invalid("item", "is required")
	case d.Amount < 0:
		return invalid("amount", "must not be negative")
	}
	return nil
}

type YearFilter struct {
	Year *int
}

type NoFilter struct{}

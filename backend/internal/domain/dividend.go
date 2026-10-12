package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DividendSourceType names the page an asset lives on. PVD is not a source: it
// pays nothing out in cash.
type DividendSourceType string

const (
	SourceBroker  DividendSourceType = "broker"
	SourceFund    DividendSourceType = "fund"
	SourceSavings DividendSourceType = "savings"
	SourceInsurer DividendSourceType = "insurer"
)

var DividendSourceTypes = []DividendSourceType{SourceBroker, SourceFund, SourceSavings, SourceInsurer}

// DividendSource points at one asset, written "broker:1" at the API.
type DividendSource struct {
	Type DividendSourceType
	ID   int64
}

func (s DividendSource) String() string { return fmt.Sprintf("%s:%d", s.Type, s.ID) }

func ParseDividendSource(s string) (DividendSource, error) {
	kind, id, ok := strings.Cut(s, ":")
	n, err := strconv.ParseInt(id, 10, 64)
	src := DividendSource{Type: DividendSourceType(kind), ID: n}
	if !ok || err != nil || src.validate() != nil {
		return DividendSource{}, invalid("source", "must look like broker:1, fund:2, savings:3 or insurer:4")
	}
	return src, nil
}

func (s DividendSource) validate() error {
	for _, t := range DividendSourceTypes {
		if s.Type == t && s.ID > 0 {
			return nil
		}
	}
	return invalid("source", "is required")
}

// DividendPayment is cash an asset paid out: a stock or fund dividend, savings
// interest, or an insurance payout.
type DividendPayment struct {
	ID             int64
	Source         DividendSource
	SourceName     string
	PaidOn         time.Time
	Gross          Money
	WithholdingTax Money
	Note           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (d *DividendPayment) Validate() error {
	d.Note = strings.TrimSpace(d.Note)
	switch {
	case d.Source.validate() != nil:
		return d.Source.validate()
	case d.PaidOn.IsZero():
		return invalid("paid_on", "is required")
	case d.Gross <= 0:
		return invalid("gross", "must be greater than 0")
	case d.WithholdingTax < 0 || d.WithholdingTax > d.Gross:
		return invalid("withholding_tax", "must be between 0 and the gross amount")
	}
	return nil
}

func (d DividendPayment) Net() Money { return d.Gross - d.WithholdingTax }

// DividendSourceYear totals one source's payments in a year. Months lists the
// calendar months that paid, which shows the rhythm: monthly, twice a year, yearly.
type DividendSourceYear struct {
	Source         DividendSource
	SourceName     string
	AssetValue     Money
	Payments       int
	Months         []int
	Gross          Money
	WithholdingTax Money
}

func (s DividendSourceYear) Net() Money { return s.Gross - s.WithholdingTax }

// Yield is the year's net payout over the asset's current value.
func (s DividendSourceYear) Yield() *float64 { return Ratio(s.Net(), s.AssetValue) }

type DividendYear struct {
	Year           int
	Sources        []DividendSourceYear
	Payments       int
	Gross          Money
	WithholdingTax Money
}

func (y DividendYear) Net() Money { return y.Gross - y.WithholdingTax }

func SummarizeDividends(year int, sources []DividendSourceYear) DividendYear {
	y := DividendYear{Year: year, Sources: sources}
	for _, s := range sources {
		y.Payments += s.Payments
		y.Gross += s.Gross
		y.WithholdingTax += s.WithholdingTax
	}
	return y
}

package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func noFilter(fiber.Ctx) (domain.NoFilter, error) { return domain.NoFilter{}, nil }

func yearFilter(c fiber.Ctx) (domain.YearFilter, error) {
	year, err := queryInt[int](c, "year")
	return domain.YearFilter{Year: year}, err
}

const dateLayout = "2006-01-02"

func parseDate(field string, s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, &domain.ValidationError{Field: field, Message: "must be YYYY-MM-DD"}
	}
	return &t, nil
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

type brokerRequest struct {
	Name         string       `json:"name"`
	CurrentValue domain.Money `json:"current_value"`
	Note         string       `json:"note"`
}

type brokerResponse struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	CurrentValue domain.Money `json:"current_value"`
	Note         string       `json:"note"`
	Deposits     domain.Money `json:"deposits"`
	Withdrawals  domain.Money `json:"withdrawals"`
	NetCapital   domain.Money `json:"net_capital"`
	UnrealisedPL domain.Money `json:"unrealised_pl"`
	PLPct        *float64     `json:"pl_pct"`
	timestamps
}

func brokerResource(svc *usecase.BrokerService) resource[*domain.Broker, domain.BrokerSummary, domain.NoFilter] {
	return resource[*domain.Broker, domain.BrokerSummary, domain.NoFilter]{
		svc: svc,
		decode: decoder(func(r brokerRequest) (*domain.Broker, error) {
			return &domain.Broker{Name: r.Name, CurrentValue: r.CurrentValue, Note: r.Note}, nil
		}),
		filter: noFilter,
		present: func(b domain.BrokerSummary) any {
			return brokerResponse{
				ID: b.ID, Name: b.Name, CurrentValue: b.CurrentValue, Note: b.Note,
				Deposits: b.Deposits, Withdrawals: b.Withdrawals, NetCapital: b.NetCapital(),
				UnrealisedPL: b.UnrealisedPL(), PLPct: b.PLPercent(),
				timestamps: timestamps{b.CreatedAt, b.UpdatedAt},
			}
		},
	}
}

type brokerTxnRequest struct {
	BrokerID int64        `json:"broker_id"`
	TxnDate  *string      `json:"txn_date"`
	Type     string       `json:"type"`
	Amount   domain.Money `json:"amount"`
	Note     string       `json:"note"`
}

type brokerTxnResponse struct {
	ID         int64        `json:"id"`
	BrokerID   int64        `json:"broker_id"`
	BrokerName string       `json:"broker_name"`
	TxnDate    *string      `json:"txn_date"`
	Type       string       `json:"type"`
	Amount     domain.Money `json:"amount"`
	Note       string       `json:"note"`
	timestamps
}

func brokerTxnResource(svc *usecase.BrokerTransactionService) resource[*domain.BrokerTransaction, domain.BrokerTransaction, domain.BrokerTransactionFilter] {
	return resource[*domain.BrokerTransaction, domain.BrokerTransaction, domain.BrokerTransactionFilter]{
		svc: svc,
		decode: decoder(func(r brokerTxnRequest) (*domain.BrokerTransaction, error) {
			date, err := parseDate("txn_date", r.TxnDate)
			if err != nil {
				return nil, err
			}
			return &domain.BrokerTransaction{
				BrokerID: r.BrokerID, Date: date, Type: domain.TxnType(r.Type), Amount: r.Amount, Note: r.Note,
			}, nil
		}),
		filter: func(c fiber.Ctx) (domain.BrokerTransactionFilter, error) {
			id, err := queryInt[int64](c, "broker_id")
			return domain.BrokerTransactionFilter{BrokerID: id}, err
		},
		present: func(t domain.BrokerTransaction) any {
			return brokerTxnResponse{
				ID: t.ID, BrokerID: t.BrokerID, BrokerName: t.BrokerName, TxnDate: formatDate(t.Date),
				Type: string(t.Type), Amount: t.Amount, Note: t.Note,
				timestamps: timestamps{t.CreatedAt, t.UpdatedAt},
			}
		},
	}
}

type fundRequest struct {
	Name        string       `json:"name"`
	Category    string       `json:"category"`
	Cost        domain.Money `json:"cost"`
	MarketValue domain.Money `json:"market_value"`
	Note        string       `json:"note"`
}

type fundResponse struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Category     string       `json:"category"`
	Cost         domain.Money `json:"cost"`
	MarketValue  domain.Money `json:"market_value"`
	UnrealisedPL domain.Money `json:"unrealised_pl"`
	PLPct        *float64     `json:"pl_pct"`
	Note         string       `json:"note"`
	timestamps
}

func fundResource(svc *usecase.FundService) resource[*domain.Fund, domain.Fund, domain.NoFilter] {
	return resource[*domain.Fund, domain.Fund, domain.NoFilter]{
		svc: svc,
		decode: decoder(func(r fundRequest) (*domain.Fund, error) {
			return &domain.Fund{Name: r.Name, Category: r.Category, Cost: r.Cost, MarketValue: r.MarketValue, Note: r.Note}, nil
		}),
		filter: noFilter,
		present: func(f domain.Fund) any {
			return fundResponse{
				ID: f.ID, Name: f.Name, Category: f.Category, Cost: f.Cost, MarketValue: f.MarketValue,
				UnrealisedPL: f.UnrealisedPL(), PLPct: f.PLPercent(), Note: f.Note,
				timestamps: timestamps{f.CreatedAt, f.UpdatedAt},
			}
		},
	}
}

type pvdRequest struct {
	Year                int          `json:"year"`
	SelfContribution    domain.Money `json:"self_contribution"`
	CompanyContribution domain.Money `json:"company_contribution"`
	SelfProfit          domain.Money `json:"self_profit"`
	CompanyProfit       domain.Money `json:"company_profit"`
	FundAllocation      string       `json:"fund_allocation"`
}

type pvdResponse struct {
	ID int64 `json:"id"`
	pvdRequest
	TotalContribution domain.Money `json:"total_contribution"`
	TotalProfit       domain.Money `json:"total_profit"`
	Value             domain.Money `json:"value"`
	ReturnPct         *float64     `json:"return_pct"`
	timestamps
}

func pvdResource(svc *usecase.PVDService) resource[*domain.PVDYear, domain.PVDYear, domain.NoFilter] {
	return resource[*domain.PVDYear, domain.PVDYear, domain.NoFilter]{
		svc: svc,
		decode: decoder(func(r pvdRequest) (*domain.PVDYear, error) {
			return &domain.PVDYear{
				Year: r.Year, SelfContribution: r.SelfContribution, CompanyContribution: r.CompanyContribution,
				SelfProfit: r.SelfProfit, CompanyProfit: r.CompanyProfit, FundAllocation: r.FundAllocation,
			}, nil
		}),
		filter: noFilter,
		present: func(p domain.PVDYear) any {
			return pvdResponse{
				ID:   p.ID,
				Year: p.Year, SelfContribution: p.SelfContribution, CompanyContribution: p.CompanyContribution,
				SelfProfit: p.SelfProfit, CompanyProfit: p.CompanyProfit, FundAllocation: p.FundAllocation,
				TotalContribution: p.TotalContribution(), TotalProfit: p.TotalProfit(),
				Value: p.Value(), ReturnPct: p.ReturnPercent(),
				timestamps: timestamps{p.CreatedAt, p.UpdatedAt},
			}
		},
	}
}

type savingsRequest struct {
	Bank       string             `json:"bank"`
	Balance    domain.Money       `json:"balance"`
	InterestBP domain.BasisPoints `json:"interest_bp"`
	Note       string             `json:"note"`
}

type savingsResponse struct {
	ID int64 `json:"id"`
	savingsRequest
	InterestPerYear domain.Money `json:"interest_per_year"`
	timestamps
}

func savingsResource(svc *usecase.SavingsService) resource[*domain.SavingsAccount, domain.SavingsAccount, domain.NoFilter] {
	return resource[*domain.SavingsAccount, domain.SavingsAccount, domain.NoFilter]{
		svc: svc,
		decode: decoder(func(r savingsRequest) (*domain.SavingsAccount, error) {
			return &domain.SavingsAccount{Bank: r.Bank, Balance: r.Balance, InterestRate: r.InterestBP, Note: r.Note}, nil
		}),
		filter: noFilter,
		present: func(s domain.SavingsAccount) any {
			return savingsResponse{
				ID:   s.ID,
				Bank: s.Bank, Balance: s.Balance, InterestBP: s.InterestRate, Note: s.Note,
				InterestPerYear: s.InterestPerYear(),
				timestamps:      timestamps{s.CreatedAt, s.UpdatedAt},
			}
		},
	}
}

type insurerRequest struct {
	Company string `json:"company"`
	Note    string `json:"note"`
}

type insurerResponse struct {
	ID int64 `json:"id"`
	insurerRequest
	Payments  int          `json:"payments"`
	TotalPaid domain.Money `json:"total_paid"`
	timestamps
}

func insurerResource(svc *usecase.InsurerService) resource[*domain.Insurer, domain.InsurerSummary, domain.NoFilter] {
	return resource[*domain.Insurer, domain.InsurerSummary, domain.NoFilter]{
		svc: svc,
		decode: decoder(func(r insurerRequest) (*domain.Insurer, error) {
			return &domain.Insurer{Company: r.Company, Note: r.Note}, nil
		}),
		filter: noFilter,
		present: func(i domain.InsurerSummary) any {
			return insurerResponse{
				ID: i.ID, Company: i.Company, Note: i.Note,
				Payments: i.Payments, TotalPaid: i.TotalPaid,
				timestamps: timestamps{i.CreatedAt, i.UpdatedAt},
			}
		},
	}
}

type paymentRequest struct {
	InsurerID int64        `json:"insurer_id"`
	Year      *int         `json:"year"`
	Premium   domain.Money `json:"premium"`
	Note      string       `json:"note"`
}

type paymentResponse struct {
	ID int64 `json:"id"`
	paymentRequest
	Company   string `json:"company"`
	PaymentNo int    `json:"payment_no"`
	timestamps
}

func paymentResource(svc *usecase.InsurancePaymentService) resource[*domain.InsurancePayment, domain.InsurancePayment, domain.InsurancePaymentFilter] {
	return resource[*domain.InsurancePayment, domain.InsurancePayment, domain.InsurancePaymentFilter]{
		svc: svc,
		decode: decoder(func(r paymentRequest) (*domain.InsurancePayment, error) {
			return &domain.InsurancePayment{InsurerID: r.InsurerID, Year: r.Year, Premium: r.Premium, Note: r.Note}, nil
		}),
		filter: func(c fiber.Ctx) (domain.InsurancePaymentFilter, error) {
			insurer, err := queryInt[int64](c, "insurer_id")
			if err != nil {
				return domain.InsurancePaymentFilter{}, err
			}
			year, err := queryInt[int](c, "year")
			return domain.InsurancePaymentFilter{InsurerID: insurer, Year: year}, err
		},
		present: func(p domain.InsurancePayment) any {
			return paymentResponse{
				ID:        p.ID,
				InsurerID: p.InsurerID, Year: p.Year, Premium: p.Premium, Note: p.Note,
				Company: p.Company, PaymentNo: p.PaymentNo,
				timestamps: timestamps{p.CreatedAt, p.UpdatedAt},
			}
		},
	}
}

type salaryRequest struct {
	Year           int          `json:"year"`
	Month          int          `json:"month"`
	Salary         domain.Money `json:"salary"`
	Bonus          domain.Money `json:"bonus"`
	WithholdingTax domain.Money `json:"withholding_tax"`
	PVD            domain.Money `json:"pvd"`
	SSO            domain.Money `json:"sso"`
}

type salaryResponse struct {
	ID int64 `json:"id"`
	salaryRequest
	timestamps
}

func salaryResource(svc *usecase.SalaryService) resource[*domain.SalaryMonth, domain.SalaryMonth, domain.YearFilter] {
	return resource[*domain.SalaryMonth, domain.SalaryMonth, domain.YearFilter]{
		svc: svc,
		decode: decoder(func(r salaryRequest) (*domain.SalaryMonth, error) {
			return &domain.SalaryMonth{
				Year: r.Year, Month: r.Month, Salary: r.Salary, Bonus: r.Bonus,
				WithholdingTax: r.WithholdingTax, PVD: r.PVD, SSO: r.SSO,
			}, nil
		}),
		filter: yearFilter,
		present: func(s domain.SalaryMonth) any {
			return salaryResponse{
				ID:   s.ID,
				Year: s.Year, Month: s.Month, Salary: s.Salary, Bonus: s.Bonus,
				WithholdingTax: s.WithholdingTax, PVD: s.PVD, SSO: s.SSO,
				timestamps: timestamps{s.CreatedAt, s.UpdatedAt},
			}
		},
	}
}

type deductionRequest struct {
	Year   int          `json:"year"`
	Item   string       `json:"item"`
	Amount domain.Money `json:"amount"`
}

type deductionResponse struct {
	ID int64 `json:"id"`
	deductionRequest
	timestamps
}

func deductionResource(svc *usecase.TaxDeductionService) resource[*domain.TaxDeduction, domain.TaxDeduction, domain.YearFilter] {
	return resource[*domain.TaxDeduction, domain.TaxDeduction, domain.YearFilter]{
		svc: svc,
		decode: decoder(func(r deductionRequest) (*domain.TaxDeduction, error) {
			return &domain.TaxDeduction{Year: r.Year, Item: r.Item, Amount: r.Amount}, nil
		}),
		filter: yearFilter,
		present: func(d domain.TaxDeduction) any {
			return deductionResponse{
				ID:   d.ID,
				Year: d.Year, Item: d.Item, Amount: d.Amount,
				timestamps: timestamps{d.CreatedAt, d.UpdatedAt},
			}
		},
	}
}

type dividendRequest struct {
	Source         string       `json:"source"`
	PaidOn         *string      `json:"paid_on"`
	Gross          domain.Money `json:"gross"`
	WithholdingTax domain.Money `json:"withholding_tax"`
	Note           string       `json:"note"`
}

type dividendResponse struct {
	ID             int64        `json:"id"`
	Source         string       `json:"source"`
	SourceType     string       `json:"source_type"`
	SourceID       int64        `json:"source_id"`
	SourceName     string       `json:"source_name"`
	PaidOn         *string      `json:"paid_on"`
	Gross          domain.Money `json:"gross"`
	WithholdingTax domain.Money `json:"withholding_tax"`
	Net            domain.Money `json:"net"`
	Note           string       `json:"note"`
	timestamps
}

func dividendResource(svc *usecase.DividendService) resource[*domain.DividendPayment, domain.DividendPayment, domain.YearFilter] {
	return resource[*domain.DividendPayment, domain.DividendPayment, domain.YearFilter]{
		svc: svc,
		decode: decoder(func(r dividendRequest) (*domain.DividendPayment, error) {
			source, err := domain.ParseDividendSource(r.Source)
			if err != nil {
				return nil, err
			}
			paidOn, err := parseDate("paid_on", r.PaidOn)
			if err != nil {
				return nil, err
			}
			d := &domain.DividendPayment{Source: source, Gross: r.Gross, WithholdingTax: r.WithholdingTax, Note: r.Note}
			if paidOn != nil {
				d.PaidOn = *paidOn
			}
			return d, nil
		}),
		filter: yearFilter,
		present: func(d domain.DividendPayment) any {
			return dividendResponse{
				ID: d.ID, Source: d.Source.String(), SourceType: string(d.Source.Type), SourceID: d.Source.ID,
				SourceName: d.SourceName, PaidOn: formatDate(&d.PaidOn), Gross: d.Gross,
				WithholdingTax: d.WithholdingTax, Net: d.Net(), Note: d.Note,
				timestamps: timestamps{d.CreatedAt, d.UpdatedAt},
			}
		},
	}
}

type dividendSourceResponse struct {
	Source         string       `json:"source"`
	SourceType     string       `json:"source_type"`
	SourceID       int64        `json:"source_id"`
	SourceName     string       `json:"source_name"`
	AssetValue     domain.Money `json:"asset_value"`
	Payments       int          `json:"payments"`
	Months         []int        `json:"months"`
	Gross          domain.Money `json:"gross"`
	WithholdingTax domain.Money `json:"withholding_tax"`
	Net            domain.Money `json:"net"`
	YieldPct       *float64     `json:"yield_pct"`
}

type dividendYearResponse struct {
	Year           int                      `json:"year"`
	Sources        []dividendSourceResponse `json:"sources"`
	Payments       int                      `json:"payments"`
	Gross          domain.Money             `json:"gross"`
	WithholdingTax domain.Money             `json:"withholding_tax"`
	Net            domain.Money             `json:"net"`
}

// dividendSummary answers GET /dividends/summary?year=YYYY (default: this year).
func dividendSummary(svc *usecase.DividendSummaryService) fiber.Handler {
	return func(c fiber.Ctx) error {
		year, err := queryInt[int](c, "year")
		if err != nil {
			return err
		}
		if year == nil {
			now := time.Now().Year()
			year = &now
		}
		y, err := svc.Year(c.Context(), *year)
		if err != nil {
			return err
		}
		resp := dividendYearResponse{
			Year: y.Year, Sources: make([]dividendSourceResponse, len(y.Sources)),
			Payments: y.Payments, Gross: y.Gross, WithholdingTax: y.WithholdingTax, Net: y.Net(),
		}
		for i, s := range y.Sources {
			resp.Sources[i] = dividendSourceResponse{
				Source: s.Source.String(), SourceType: string(s.Source.Type), SourceID: s.Source.ID,
				SourceName: s.SourceName, AssetValue: s.AssetValue, Payments: s.Payments, Months: s.Months,
				Gross: s.Gross, WithholdingTax: s.WithholdingTax, Net: s.Net(), YieldPct: s.Yield(),
			}
		}
		return c.JSON(resp)
	}
}

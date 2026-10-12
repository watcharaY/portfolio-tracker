package domain

type AssetClass string

const (
	AssetStocks    AssetClass = "stocks"
	AssetFunds     AssetClass = "funds"
	AssetPVD       AssetClass = "pvd"
	AssetSavings   AssetClass = "savings"
	AssetInsurance AssetClass = "insurance"
)

type AssetClassSummary struct {
	Class      AssetClass
	Label      string
	Value      Money
	Cost       Money
	PL         Money
	PLPercent  *float64
	Allocation *float64
}

type PortfolioSummary struct {
	Classes                   []AssetClassSummary
	Total                     AssetClassSummary
	EstSavingsInterestPerYear Money
}

type Holdings struct {
	Brokers   []BrokerSummary
	Funds     []Fund
	PVD       []PVDYear
	Savings   []SavingsAccount
	Insurance []InsurerSummary
}

func Summarize(h Holdings) PortfolioSummary {
	var stocks, funds, pvd, savings, insurance AssetClassSummary
	stocks = AssetClassSummary{Class: AssetStocks, Label: "Stocks (Brokerage)"}
	for _, b := range h.Brokers {
		stocks.Value += b.CurrentValue
		stocks.Cost += b.NetCapital()
	}
	funds = AssetClassSummary{Class: AssetFunds, Label: "Mutual Funds"}
	for _, f := range h.Funds {
		funds.Value += f.MarketValue
		funds.Cost += f.Cost
	}
	pvd = AssetClassSummary{Class: AssetPVD, Label: "Provident Fund"}
	for _, p := range h.PVD {
		pvd.Value += p.Value()
		pvd.Cost += p.TotalContribution()
	}
	var interest Money
	savings = AssetClassSummary{Class: AssetSavings, Label: "Savings"}
	for _, s := range h.Savings {
		savings.Value += s.Balance
		interest += s.InterestPerYear()
	}
	savings.Cost = savings.Value
	insurance = AssetClassSummary{Class: AssetInsurance, Label: "Insurance (premiums)"}
	for _, i := range h.Insurance {
		insurance.Value += i.TotalPaid
	}
	insurance.Cost = insurance.Value

	classes := []AssetClassSummary{stocks, funds, pvd, savings, insurance}
	total := AssetClassSummary{Label: "Total"}
	for _, c := range classes {
		total.Value += c.Value
		total.Cost += c.Cost
	}
	for i := range classes {
		finish(&classes[i], total.Value)
	}
	finish(&total, total.Value)

	return PortfolioSummary{Classes: classes, Total: total, EstSavingsInterestPerYear: interest}
}

func finish(c *AssetClassSummary, totalValue Money) {
	c.PL = c.Value - c.Cost
	c.PLPercent = Ratio(c.PL, c.Cost)
	c.Allocation = Ratio(c.Value, totalValue)
}

package domain

import (
	"strings"
	"time"
)

type Broker struct {
	ID           int64
	Name         string
	CurrentValue Money
	Note         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (b *Broker) Validate() error {
	b.Name = strings.TrimSpace(b.Name)
	switch {
	case b.Name == "":
		return invalid("name", "is required")
	case b.CurrentValue < 0:
		return invalid("current_value", "must not be negative")
	}
	return nil
}

type BrokerSummary struct {
	Broker
	Deposits    Money
	Withdrawals Money
}

func (s BrokerSummary) NetCapital() Money   { return s.Deposits - s.Withdrawals }
func (s BrokerSummary) UnrealisedPL() Money { return s.CurrentValue - s.NetCapital() }

func (s BrokerSummary) PLPercent() *float64 {
	if s.NetCapital() <= 0 {
		return nil
	}
	return Ratio(s.UnrealisedPL(), s.NetCapital())
}

type TxnType string

const (
	TxnDeposit  TxnType = "deposit"
	TxnWithdraw TxnType = "withdraw"
)

func (t TxnType) Valid() bool { return t == TxnDeposit || t == TxnWithdraw }

type BrokerTransaction struct {
	ID         int64
	BrokerID   int64
	BrokerName string
	Date       *time.Time
	Type       TxnType
	Amount     Money
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (t *BrokerTransaction) Validate() error {
	switch {
	case t.BrokerID <= 0:
		return invalid("broker_id", "is required")
	case !t.Type.Valid():
		return invalid("type", "must be deposit or withdraw")
	case t.Amount <= 0:
		return invalid("amount", "must be greater than 0")
	}
	return nil
}

type BrokerTransactionFilter struct {
	BrokerID *int64
}

package usecase

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type (
	BrokerService            = CRUD[*domain.Broker, domain.BrokerSummary, domain.NoFilter]
	BrokerTransactionService = CRUD[*domain.BrokerTransaction, domain.BrokerTransaction, domain.BrokerTransactionFilter]
	FundService              = CRUD[*domain.Fund, domain.Fund, domain.NoFilter]
	PVDService               = CRUD[*domain.PVDYear, domain.PVDYear, domain.NoFilter]
	SavingsService           = CRUD[*domain.SavingsAccount, domain.SavingsAccount, domain.NoFilter]
	InsurerService           = CRUD[*domain.Insurer, domain.InsurerSummary, domain.NoFilter]
	InsurancePaymentService  = CRUD[*domain.InsurancePayment, domain.InsurancePayment, domain.InsurancePaymentFilter]
	SalaryService            = CRUD[*domain.SalaryMonth, domain.SalaryMonth, domain.YearFilter]
	TaxDeductionService      = CRUD[*domain.TaxDeduction, domain.TaxDeduction, domain.YearFilter]
)

type CRUD[W Validatable, R any, F any] struct {
	repo Repository[W, R, F]
}

func NewCRUD[W Validatable, R any, F any](repo Repository[W, R, F]) *CRUD[W, R, F] {
	return &CRUD[W, R, F]{repo: repo}
}

func (s *CRUD[W, R, F]) List(ctx context.Context, filter F) ([]R, error) {
	return s.repo.List(ctx, filter)
}

func (s *CRUD[W, R, F]) Get(ctx context.Context, id int64) (R, error) {
	return s.repo.Get(ctx, id)
}

func (s *CRUD[W, R, F]) Create(ctx context.Context, entity W) (R, error) {
	var zero R
	if err := entity.Validate(); err != nil {
		return zero, err
	}
	id, err := s.repo.Create(ctx, entity)
	if err != nil {
		return zero, err
	}
	return s.repo.Get(ctx, id)
}

func (s *CRUD[W, R, F]) Update(ctx context.Context, id int64, entity W) (R, error) {
	var zero R
	if err := entity.Validate(); err != nil {
		return zero, err
	}
	if err := s.repo.Update(ctx, id, entity); err != nil {
		return zero, err
	}
	return s.repo.Get(ctx, id)
}

func (s *CRUD[W, R, F]) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

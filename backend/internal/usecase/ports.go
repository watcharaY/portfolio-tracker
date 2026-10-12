package usecase

import (
	"context"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type Validatable interface {
	Validate() error
}

type Repository[W Validatable, R any, F any] interface {
	List(ctx context.Context, filter F) ([]R, error)
	Get(ctx context.Context, id int64) (R, error)
	Create(ctx context.Context, entity W) (int64, error)
	Update(ctx context.Context, id int64, entity W) error
	Delete(ctx context.Context, id int64) error
}

type (
	BrokerRepository            = Repository[*domain.Broker, domain.BrokerSummary, domain.NoFilter]
	BrokerTransactionRepository = Repository[*domain.BrokerTransaction, domain.BrokerTransaction, domain.BrokerTransactionFilter]
	FundRepository              = Repository[*domain.Fund, domain.Fund, domain.NoFilter]
	PVDRepository               = Repository[*domain.PVDYear, domain.PVDYear, domain.NoFilter]
	SavingsRepository           = Repository[*domain.SavingsAccount, domain.SavingsAccount, domain.NoFilter]
	InsurerRepository           = Repository[*domain.Insurer, domain.InsurerSummary, domain.NoFilter]
	InsurancePaymentRepository  = Repository[*domain.InsurancePayment, domain.InsurancePayment, domain.InsurancePaymentFilter]
	SalaryRepository            = Repository[*domain.SalaryMonth, domain.SalaryMonth, domain.YearFilter]
	TaxDeductionRepository      = Repository[*domain.TaxDeduction, domain.TaxDeduction, domain.YearFilter]
)

type UserRepository interface {
	Count(ctx context.Context) (int, error)
	GetByID(ctx context.Context, id int64) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	Create(ctx context.Context, user domain.User) (int64, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string, mustChange bool) error
}

type SessionRepository interface {
	Create(ctx context.Context, s domain.Session) error
	GetValid(ctx context.Context, tokenHash string, now time.Time) (domain.Session, error)
	Delete(ctx context.Context, tokenHash string) error
	DeleteOthers(ctx context.Context, userID int64, keepTokenHash string) error
	DeleteExpired(ctx context.Context, now time.Time) error
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type TokenIssuer interface {
	New() (string, error)
	Hash(token string) string
}

type Clock func() time.Time

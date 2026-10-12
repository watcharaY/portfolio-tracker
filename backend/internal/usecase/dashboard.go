package usecase

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type DashboardService struct {
	brokers  BrokerRepository
	funds    FundRepository
	pvd      PVDRepository
	savings  SavingsRepository
	insurers InsurerRepository
}

func NewDashboardService(
	brokers BrokerRepository, funds FundRepository, pvd PVDRepository,
	savings SavingsRepository, insurers InsurerRepository,
) *DashboardService {
	return &DashboardService{brokers: brokers, funds: funds, pvd: pvd, savings: savings, insurers: insurers}
}

func (s *DashboardService) Summary(ctx context.Context) (domain.PortfolioSummary, error) {
	var h domain.Holdings
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { h.Brokers, err = s.brokers.List(ctx, domain.NoFilter{}); return })
	g.Go(func() (err error) { h.Funds, err = s.funds.List(ctx, domain.NoFilter{}); return })
	g.Go(func() (err error) { h.PVD, err = s.pvd.List(ctx, domain.NoFilter{}); return })
	g.Go(func() (err error) { h.Savings, err = s.savings.List(ctx, domain.NoFilter{}); return })
	g.Go(func() (err error) { h.Insurance, err = s.insurers.List(ctx, domain.NoFilter{}); return })
	if err := g.Wait(); err != nil {
		return domain.PortfolioSummary{}, err
	}
	return domain.Summarize(h), nil
}

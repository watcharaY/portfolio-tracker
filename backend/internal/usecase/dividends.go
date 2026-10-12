package usecase

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type DividendSummaryService struct{ repo DividendSummaryRepository }

func NewDividendSummaryService(repo DividendSummaryRepository) *DividendSummaryService {
	return &DividendSummaryService{repo: repo}
}

func (s *DividendSummaryService) Year(ctx context.Context, year int) (domain.DividendYear, error) {
	sources, err := s.repo.Summary(ctx, year)
	if err != nil {
		return domain.DividendYear{}, err
	}
	return domain.SummarizeDividends(year, sources), nil
}

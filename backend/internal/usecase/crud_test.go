package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type memFunds struct {
	rows   map[int64]domain.Fund
	nextID int64
	writes int
}

func (m *memFunds) List(context.Context, domain.NoFilter) ([]domain.Fund, error) {
	out := make([]domain.Fund, 0, len(m.rows))
	for _, f := range m.rows {
		out = append(out, f)
	}
	return out, nil
}
func (m *memFunds) Get(_ context.Context, id int64) (domain.Fund, error) {
	f, ok := m.rows[id]
	if !ok {
		return f, domain.ErrNotFound
	}
	return f, nil
}
func (m *memFunds) Create(_ context.Context, f *domain.Fund) (int64, error) {
	m.writes++
	m.nextID++
	f.ID = m.nextID
	m.rows[f.ID] = *f
	return f.ID, nil
}
func (m *memFunds) Update(_ context.Context, id int64, f *domain.Fund) error {
	m.writes++
	if _, ok := m.rows[id]; !ok {
		return domain.ErrNotFound
	}
	f.ID = id
	m.rows[id] = *f
	return nil
}
func (m *memFunds) Delete(_ context.Context, id int64) error { delete(m.rows, id); return nil }

func TestCRUD_ValidatesBeforePersisting(t *testing.T) {
	repo := &memFunds{rows: map[int64]domain.Fund{}}
	svc := usecase.NewCRUD[*domain.Fund, domain.Fund, domain.NoFilter](repo)
	ctx := context.Background()

	var ve *domain.ValidationError
	if _, err := svc.Create(ctx, &domain.Fund{Name: "X", Category: "Crypto"}); !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if repo.writes != 0 {
		t.Fatal("invalid entity must not reach the repository")
	}

	created, err := svc.Create(ctx, &domain.Fund{Name: "KT-JEPQ", Category: "Equity", Cost: 10_000, MarketValue: 10_483})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.UnrealisedPL() != 483 {
		t.Fatalf("created = %+v", created)
	}

	updated, err := svc.Update(ctx, created.ID, &domain.Fund{Name: "KT-JEPQ", Category: "Equity", Cost: 10_000, MarketValue: 9_000})
	if err != nil || updated.UnrealisedPL() != -1_000 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if _, err := svc.Update(ctx, 999, &domain.Fund{Name: "A", Category: "Gold"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("update missing: got %v", err)
	}
}

package postgres_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/security"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type expectedClass struct {
	Value         domain.Money `json:"value"`
	Cost          domain.Money `json:"cost"`
	PL            domain.Money `json:"pl"`
	PLPct         float64      `json:"plPct"`
	AllocationPct float64      `json:"allocationPct"`
}

type expectedUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type e2eExpected struct {
	Dashboard struct {
		Total                     expectedClass            `json:"total"`
		Classes                   map[string]expectedClass `json:"classes"`
		EstSavingsInterestPerYear domain.Money             `json:"estSavingsInterestPerYear"`
	} `json:"dashboard"`
	Users struct {
		ForcedChange expectedUser `json:"forcedChange"`
		Ready        expectedUser `json:"ready"`
	} `json:"users"`
}

func loadE2ESeed(t *testing.T) (*postgres.DB, e2eExpected) {
	t.Helper()
	db := openDB(t)
	seed, err := os.ReadFile("../../../testdata/e2e-seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(t, string(seed))

	raw, err := os.ReadFile("../../../testdata/e2e-expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var want e2eExpected
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	return db, want
}

func TestE2ESeed_DashboardMatchesExpected(t *testing.T) {
	db, want := loadE2ESeed(t)
	svc := usecase.NewDashboardService(
		postgres.NewBrokerRepo(db), postgres.NewFundRepo(db), postgres.NewPVDRepo(db),
		postgres.NewSavingsRepo(db), postgres.NewInsurerRepo(db),
	)
	got, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	check := func(name string, got domain.AssetClassSummary, want expectedClass) {
		t.Helper()
		if got.Value != want.Value || got.Cost != want.Cost || got.PL != want.PL {
			t.Errorf("%s: value/cost/pl = %d/%d/%d, want %d/%d/%d",
				name, got.Value, got.Cost, got.PL, want.Value, want.Cost, want.PL)
		}
		for field, pair := range map[string][2]any{
			"plPct":         {got.PLPercent, want.PLPct},
			"allocationPct": {got.Allocation, want.AllocationPct},
		} {
			g, w := pair[0].(*float64), pair[1].(float64)
			if g == nil || math.Abs(*g-w) > 5e-7 {
				t.Errorf("%s: %s = %v, want %v", name, field, g, w)
			}
		}
	}
	check("total", got.Total, want.Dashboard.Total)
	if len(got.Classes) != len(want.Dashboard.Classes) {
		t.Fatalf("got %d classes, want %d", len(got.Classes), len(want.Dashboard.Classes))
	}
	for _, c := range got.Classes {
		check(string(c.Class), c, want.Dashboard.Classes[string(c.Class)])
	}
	if got.EstSavingsInterestPerYear != want.Dashboard.EstSavingsInterestPerYear {
		t.Errorf("interest = %d, want %d", got.EstSavingsInterestPerYear, want.Dashboard.EstSavingsInterestPerYear)
	}
}

func TestE2ESeed_UsersLogInWithDocumentedPasswords(t *testing.T) {
	db, want := loadE2ESeed(t)
	users := postgres.NewUserRepo(db)
	hasher := security.BcryptHasher{}

	for _, c := range []struct {
		user       expectedUser
		mustChange bool
	}{
		{want.Users.ForcedChange, true},
		{want.Users.Ready, false},
	} {
		u, err := users.GetByUsername(context.Background(), c.user.Username)
		if err != nil {
			t.Fatalf("%s: %v", c.user.Username, err)
		}
		if !hasher.Compare(u.PasswordHash, c.user.Password) {
			t.Errorf("%s: password in e2e-expected.json does not match the seeded hash", c.user.Username)
		}
		if u.MustChangePassword != c.mustChange {
			t.Errorf("%s: must_change_password = %v, want %v", c.user.Username, u.MustChangePassword, c.mustChange)
		}
	}
}

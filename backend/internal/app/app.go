package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"

	httpadapter "github.com/watcharaY/portfolio-tracker/backend/internal/adapter/http"
	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres"
	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/config"
	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/security"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

func Migrate(ctx context.Context, cfg config.Config) error {
	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := postgres.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func Run(ctx context.Context, cfg config.Config) error {
	httpCfg := httpadapter.Config{
		CookieSecure:       cfg.CookieSecure,
		CORSOrigin:         cfg.CORSOrigin,
		TrustedProxies:     cfg.TrustedProxies,
		RateLimitPerMinute: cfg.RateLimitPerMinute,
	}
	if cfg.WebDir != "" {
		httpCfg.WebUI = os.DirFS(cfg.WebDir)
		if _, err := fs.Stat(httpCfg.WebUI, "index.html"); err != nil {
			return fmt.Errorf("WEB_DIR %q: %w", cfg.WebDir, err)
		}
	}

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := postgres.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	services, err := buildServices(db, cfg)
	if err != nil {
		return err
	}
	seeded, err := services.Auth.EnsureDefaultAdmin(ctx)
	if err != nil {
		return err
	}
	if seeded {
		slog.Warn("seeded default user admin/admin — you will be asked to change the password on first login")
	}

	server := httpadapter.NewServer(httpCfg, services)

	slog.Info("listening", "addr", cfg.Addr)
	return server.Listen(cfg.Addr, fiber.ListenConfig{
		GracefulContext:       ctx,
		ShutdownTimeout:       10 * time.Second,
		DisableStartupMessage: true,
	})
}

func buildServices(db *postgres.DB, cfg config.Config) (httpadapter.Services, error) {
	brokers := postgres.NewBrokerRepo(db)
	funds := postgres.NewFundRepo(db)
	pvd := postgres.NewPVDRepo(db)
	savings := postgres.NewSavingsRepo(db)
	insurers := postgres.NewInsurerRepo(db)
	dividends := postgres.NewDividendRepo(db)

	auth, err := usecase.NewAuthService(usecase.AuthDeps{
		Users:    postgres.NewUserRepo(db),
		Sessions: postgres.NewSessionRepo(db),
		Tx:       db,
		Hasher:   security.BcryptHasher{},
		Tokens:   security.RandomTokens{},
		Clock:    time.Now,
		TTL:      cfg.SessionTTL,
	})
	if err != nil {
		return httpadapter.Services{}, err
	}

	return httpadapter.Services{
		Ready:             db.Ping,
		Auth:              auth,
		Dashboard:         usecase.NewDashboardService(brokers, funds, pvd, savings, insurers),
		Brokers:           usecase.NewCRUD(brokers),
		BrokerTxns:        usecase.NewCRUD(postgres.NewBrokerTransactionRepo(db)),
		Funds:             usecase.NewCRUD(funds),
		PVD:               usecase.NewCRUD(pvd),
		Savings:           usecase.NewCRUD(savings),
		Insurers:          usecase.NewCRUD(insurers),
		InsurancePayments: usecase.NewCRUD(postgres.NewInsurancePaymentRepo(db)),
		Salary:            usecase.NewCRUD(postgres.NewSalaryRepo(db)),
		TaxDeductions:     usecase.NewCRUD(postgres.NewTaxDeductionRepo(db)),
		Dividends:         usecase.NewCRUD(dividends),
		DividendSummary:   usecase.NewDividendSummaryService(dividends),
	}, nil
}

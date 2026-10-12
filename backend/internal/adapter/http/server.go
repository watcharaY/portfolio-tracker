package http

import (
	"io/fs"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type Config struct {
	CookieSecure bool
	CORSOrigin   string
	WebUI        fs.FS
}

type Services struct {
	Auth              *usecase.AuthService
	Dashboard         *usecase.DashboardService
	Brokers           *usecase.BrokerService
	BrokerTxns        *usecase.BrokerTransactionService
	Funds             *usecase.FundService
	PVD               *usecase.PVDService
	Savings           *usecase.SavingsService
	Insurers          *usecase.InsurerService
	InsurancePayments *usecase.InsurancePaymentService
	Salary            *usecase.SalaryService
	TaxDeductions     *usecase.TaxDeductionService
}

func NewServer(cfg Config, s Services) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Portfolio Tracker",
		BodyLimit:    1 << 20,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New(), logger.New(), helmet.New())
	if cfg.CORSOrigin != "" {
		app.Use(cors.New(cors.Config{AllowOrigins: []string{cfg.CORSOrigin}, AllowCredentials: true}))
	}

	cookies := sessionCookies{secure: cfg.CookieSecure}
	authH := &authHandler{auth: s.Auth, cookies: cookies}
	requireAuth := requireSession(s.Auth, cookies)

	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })
	api.Post("/auth/login", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), authH.login)

	pending := api.Group("/auth", requireAuth(true))
	pending.Get("/me", authH.me)
	pending.Post("/logout", authH.logout)
	pending.Post("/password", authH.changePassword)

	priv := api.Group("", requireAuth(false))
	priv.Get("/dashboard", (&dashboardHandler{svc: s.Dashboard}).summary)
	mount(priv, "/brokers", brokerResource(s.Brokers))
	mount(priv, "/broker-transactions", brokerTxnResource(s.BrokerTxns))
	mount(priv, "/funds", fundResource(s.Funds))
	mount(priv, "/pvd-years", pvdResource(s.PVD))
	mount(priv, "/savings-accounts", savingsResource(s.Savings))
	mount(priv, "/insurers", insurerResource(s.Insurers))
	mount(priv, "/insurance-payments", paymentResource(s.InsurancePayments))
	mount(priv, "/salary-months", salaryResource(s.Salary))
	mount(priv, "/tax-deductions", deductionResource(s.TaxDeductions))
	api.Use(func(fiber.Ctx) error { return fiber.ErrNotFound })

	mountSPA(app, cfg.WebUI)
	return app
}

func mountSPA(app *fiber.App, webUI fs.FS) {
	index, err := fs.ReadFile(webUI, "index.html")
	if err != nil {
		index = []byte(`<h1>Frontend not built</h1><p>Run <code>npm run build</code> in <code>web/</code>, or use the Vite dev server on :5173.</p>`)
	}
	app.Use("/", static.New("", static.Config{FS: webUI}))
	app.Get("/*", func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/") {
			return fiber.ErrNotFound
		}
		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.Type("html").Send(index)
	})
}

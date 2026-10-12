package http

import (
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/http/openapi"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type Config struct {
	CookieSecure bool
	CORSOrigin   string
	// WebUI is the built SPA to serve next to the API. Nil serves the API only.
	WebUI fs.FS
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
	app.Use(recover.New(), logger.New(), helmet.New(helmet.Config{
		// The built SPA only loads its own hashed bundles; React's style={} goes through CSSOM, which CSP allows.
		// Vite inlines small font subsets as data: URIs.
		ContentSecurityPolicy: "default-src 'self'; img-src 'self' data:; font-src 'self' data:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'",
	}))
	if cfg.CookieSecure {
		// helmet only sends HSTS when Fiber itself terminates TLS; here TLS ends at the reverse proxy.
		app.Use(func(c fiber.Ctx) error {
			c.Set(fiber.HeaderStrictTransportSecurity, "max-age=15552000")
			return c.Next()
		})
	}
	if cfg.CORSOrigin != "" {
		app.Use(cors.New(cors.Config{AllowOrigins: []string{cfg.CORSOrigin}, AllowCredentials: true}))
	}

	cookies := sessionCookies{secure: cfg.CookieSecure}
	authH := &authHandler{auth: s.Auth, cookies: cookies}
	requireAuth := requireSession(s.Auth, cookies)

	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })
	api.Use(requireJSONBody)
	api.Post("/auth/login", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), authH.login)

	pending := api.Group("/auth", requireAuth(true))
	pending.Get("/me", authH.me)
	pending.Post("/logout", authH.logout)
	pending.Post("/password", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), authH.changePassword)

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
	mountDocs(priv)
	api.Use(func(fiber.Ctx) error { return fiber.ErrNotFound })

	if cfg.WebUI != nil {
		mountSPA(app, cfg.WebUI)
	}
	return app
}

// mountDocs serves the OpenAPI document and its Swagger UI. They sit behind the
// session like the rest of the API, so the docs are only visible once signed in.
func mountDocs(r fiber.Router) {
	r.Get("/openapi.yaml", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml")
		return c.Send(openapi.Spec)
	})
	r.Get("/docs", func(c fiber.Ctx) error { return sendDocsFile(c, "docs.html") })
	r.Get("/docs/*", func(c fiber.Ctx) error { return sendDocsFile(c, c.Params("*")) })
}

func sendDocsFile(c fiber.Ctx, name string) error {
	data, err := fs.ReadFile(openapi.UI, name)
	if err != nil {
		return fiber.ErrNotFound
	}
	return c.Type(strings.TrimPrefix(path.Ext(name), ".")).Send(data)
}

// requireJSONBody rejects POST/PUT bodies that are not application/json. Browsers
// cannot send that content type cross-site without a CORS preflight, so this backs
// up the SameSite=Strict cookie against CSRF.
func requireJSONBody(c fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodPost, fiber.MethodPut:
		if !strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEApplicationJSON) {
			return fiber.NewError(fiber.StatusUnsupportedMediaType, "Content-Type must be application/json")
		}
	}
	return c.Next()
}

// mountSPA serves the built SPA and falls back to index.html for client routes.
// index.html is no-cache so a deploy is picked up at once; the hashed assets are not.
func mountSPA(app *fiber.App, webUI fs.FS) {
	app.Use("/", static.New("", static.Config{
		FS: webUI,
		ModifyResponse: func(c fiber.Ctx) error {
			if strings.HasPrefix(string(c.Response().Header.ContentType()), fiber.MIMETextHTML) {
				c.Set(fiber.HeaderCacheControl, "no-cache")
			}
			return nil
		},
	}))
	app.Get("/*", func(c fiber.Ctx) error {
		index, err := fs.ReadFile(webUI, "index.html")
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.Type("html").Send(index)
	})
}

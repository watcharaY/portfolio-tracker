package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

const sessionCookie = "pt_session"

type userKey struct{}

type sessionCookies struct{ secure bool }

func (s sessionCookies) set(c fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name: sessionCookie, Value: token, Path: "/", Expires: expires,
		HTTPOnly: true, Secure: s.secure, SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func (s sessionCookies) clear(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HTTPOnly: true, Secure: s.secure, SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func requireSession(auth *usecase.AuthService, cookies sessionCookies) func(allowPending bool) fiber.Handler {
	return func(allowPending bool) fiber.Handler {
		return func(c fiber.Ctx) error {
			user, err := auth.Authenticate(c.Context(), c.Cookies(sessionCookie))
			if errors.Is(err, domain.ErrUnauthenticated) {
				cookies.clear(c)
			}
			if err != nil {
				return err
			}
			if user.MustChangePassword && !allowPending {
				return domain.ErrPasswordChangeRequired
			}
			c.Locals(userKey{}, user)
			return c.Next()
		}
	}
}

func currentUser(c fiber.Ctx) domain.User {
	u, _ := c.Locals(userKey{}).(domain.User)
	return u
}

type authHandler struct {
	auth    *usecase.AuthService
	cookies sessionCookies
}

type userResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func toUserResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Username: u.Username, MustChangePassword: u.MustChangePassword}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *authHandler) login(c fiber.Ctx) error {
	req, err := decodeJSON[loginRequest](c.Body())
	if err != nil {
		return err
	}
	res, err := h.auth.Login(c.Context(), usecase.LoginInput{
		Username: req.Username, Password: req.Password,
		IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		return err
	}
	h.cookies.set(c, res.Token, res.ExpiresAt)
	return c.JSON(toUserResponse(res.User))
}

func (h *authHandler) me(c fiber.Ctx) error {
	return c.JSON(toUserResponse(currentUser(c)))
}

func (h *authHandler) logout(c fiber.Ctx) error {
	if err := h.auth.Logout(c.Context(), c.Cookies(sessionCookie)); err != nil {
		return err
	}
	h.cookies.clear(c)
	return c.SendStatus(fiber.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *authHandler) changePassword(c fiber.Ctx) error {
	req, err := decodeJSON[changePasswordRequest](c.Body())
	if err != nil {
		return err
	}
	user, err := h.auth.ChangePassword(c.Context(), usecase.ChangePasswordInput{
		UserID:          currentUser(c).ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		CurrentToken:    c.Cookies(sessionCookie),
	})
	if err != nil {
		return err
	}
	return c.JSON(toUserResponse(user))
}

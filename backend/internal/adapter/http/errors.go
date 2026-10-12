package http

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type errorBody struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func errorHandler(c fiber.Ctx, err error) error {
	if ve, ok := errors.AsType[*domain.ValidationError](err); ok {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody{Error: ve.Error(), Field: ve.Field})
	}
	if fe, ok := errors.AsType[*fiber.Error](err); ok {
		return c.Status(fe.Code).JSON(errorBody{Error: fe.Message})
	}

	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrReferenceNotFound):
		status = fiber.StatusConflict
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUnauthenticated):
		status = fiber.StatusUnauthorized
	case errors.Is(err, domain.ErrPasswordChangeRequired):
		status = fiber.StatusForbidden
	}
	if status == fiber.StatusInternalServerError {
		slog.ErrorContext(c.Context(), "request failed", "request_id", requestid.FromContext(c), "method", c.Method(), "path", c.Path(), "err", err)
		return c.Status(status).JSON(errorBody{Error: "internal server error"})
	}
	return c.Status(status).JSON(errorBody{Error: err.Error()})
}

func badRequest(msg string) error { return fiber.NewError(fiber.StatusBadRequest, msg) }

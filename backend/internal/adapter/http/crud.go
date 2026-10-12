package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type resource[W usecase.Validatable, R any, F any] struct {
	svc     *usecase.CRUD[W, R, F]
	decode  func(body []byte) (W, error)
	filter  func(c fiber.Ctx) (F, error)
	present func(R) any
}

func mount[W usecase.Validatable, R any, F any](r fiber.Router, path string, res resource[W, R, F]) {
	r.Get(path, func(c fiber.Ctx) error {
		f, err := res.filter(c)
		if err != nil {
			return err
		}
		items, err := res.svc.List(c.Context(), f)
		if err != nil {
			return err
		}
		out := make([]any, len(items))
		for i, it := range items {
			out[i] = res.present(it)
		}
		return c.JSON(out)
	})

	r.Get(path+"/:id", func(c fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		item, err := res.svc.Get(c.Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(res.present(item))
	})

	r.Post(path, func(c fiber.Ctx) error {
		entity, err := res.decode(c.Body())
		if err != nil {
			return err
		}
		item, err := res.svc.Create(c.Context(), entity)
		if err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(res.present(item))
	})

	r.Put(path+"/:id", func(c fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		entity, err := res.decode(c.Body())
		if err != nil {
			return err
		}
		item, err := res.svc.Update(c.Context(), id, entity)
		if err != nil {
			return err
		}
		return c.JSON(res.present(item))
	})

	r.Delete(path+"/:id", func(c fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := res.svc.Delete(c.Context(), id); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}

func decodeJSON[T any](body []byte) (T, error) {
	var v T
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return v, &domain.ValidationError{Field: typeErr.Field, Message: "must be " + jsonKind(typeErr.Type)}
		}
		return v, badRequest("invalid JSON body: " + err.Error())
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return v, badRequest("invalid JSON body: unexpected data after object")
	}
	return v, nil
}

func jsonKind(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "an integer (money is in satang)"
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "a boolean"
	}
	return "a valid value"
}

func decoder[Req any, W any](toDomain func(Req) (W, error)) func([]byte) (W, error) {
	return func(body []byte) (W, error) {
		req, err := decodeJSON[Req](body)
		if err != nil {
			var zero W
			return zero, err
		}
		return toDomain(req)
	}
}

func pathID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, badRequest("invalid id")
	}
	return id, nil
}

func queryInt[T int | int64](c fiber.Ctx, name string) (*T, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, badRequest(name + " must be an integer")
	}
	v := T(n)
	return &v, nil
}

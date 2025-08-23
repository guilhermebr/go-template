package example

import (
	"context"
	"go-template/app/api/middleware"
	"go-template/domain/entities"

	"github.com/go-chi/chi/v5"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/example_uc.go . ExampleUseCase
type ExampleUseCase interface {
	CreateExample(ctx context.Context, example entities.Example) (string, error)
	GetExampleByID(ctx context.Context, id string) (entities.Example, error)
}

type ExampleHandler struct {
	uc ExampleUseCase
	mw *middleware.AuthMiddleware
}

func NewExampleHandler(uc ExampleUseCase, mw *middleware.AuthMiddleware) *ExampleHandler {
	return &ExampleHandler{
		uc: uc,
		mw: mw,
	}
}

func (h *ExampleHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(h.mw.RequireAuth)

	r.Post("/", h.CreateExample)
	r.Get("/{id}", h.GetExampleByID)

	return r
}

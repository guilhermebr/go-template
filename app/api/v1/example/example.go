package example

import (
	"encoding/json"
	"errors"
	"go-template/app/api/common"
	"go-template/domain"
	"go-template/domain/entities"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateExampleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type CreateExampleResponse struct {
	ID string `json:"id"`
}

// CreateExample godoc
// @Summary Create a new example
// @Description Create a new example with the given title and content
// @Tags examples
// @Accept json
// @Produce json
// @Param example body CreateExampleRequest true "Example to create"
func (h *ExampleHandler) CreateExample(w http.ResponseWriter, r *http.Request) {
	var input CreateExampleRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		common.ErrorResponse(w, r, http.StatusBadRequest, errors.New("title is required"))
		return
	}

	example := entities.Example{
		Title:   input.Title,
		Content: input.Content,
	}

	id, err := h.uc.CreateExample(r.Context(), example)
	if err != nil {
		slog.Error("failed to create example", "error", err, "input", input)
		switch {
		case errors.Is(err, domain.ErrMalformedParameters):
			common.ErrorResponse(w, r, http.StatusBadRequest, err)
			return
		case errors.Is(err, domain.ErrConflict):
			common.ErrorResponse(w, r, http.StatusConflict, err)
			return
		case errors.Is(err, domain.ErrDuplicateKey):
			common.ErrorResponse(w, r, http.StatusConflict, err)
			return
		default:
			common.UnknownErrorResponse(w, r)
			return
		}
	}

	slog.Info("example created successfully", "id", id)
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, CreateExampleResponse{ID: id})
}

// GetExampleByID godoc
// @Summary Get an example by ID
// @Description Get an example by its unique identifier
// @Tags examples
// @Accept json
// @Produce json
// @Param id path string true "Example ID"
func (h *ExampleHandler) GetExampleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		common.ErrorResponse(w, r, http.StatusBadRequest, errors.New("id is required"))
		return
	}

	example, err := h.uc.GetExampleByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get example", "error", err, "id", id)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			common.ErrorResponse(w, r, http.StatusNotFound, errors.New("example not found"))
			return
		case errors.Is(err, domain.ErrMalformedParameters):
			common.ErrorResponse(w, r, http.StatusBadRequest, err)
			return
		default:
			common.UnknownErrorResponse(w, r)
			return
		}
	}

	if example.ID == "" {
		common.ErrorResponse(w, r, http.StatusNotFound, errors.New("example not found"))
		return
	}

	slog.Info("example retrieved successfully", "id", id)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, example)
}

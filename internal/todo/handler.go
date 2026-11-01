package todo

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/2SSK/boilerplate/internal/httpx"
)

// Handler serves the todo HTTP endpoints.
type Handler struct {
	store Store
}

// NewHandler returns a Handler backed by store.
func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

// Register mounts the endpoints under prefix (for example "/api/v1/todos").
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix, h.list)
	mux.HandleFunc("POST "+prefix, h.create)
	mux.HandleFunc("PATCH "+prefix+"/{id}", h.update)
	mux.HandleFunc("DELETE "+prefix+"/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	todos, err := h.store.List(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}
	if todos == nil {
		todos = []Todo{} // encode as [] rather than null
	}
	httpx.JSON(w, http.StatusOK, todos)
}

type createRequest struct {
	Title string `json:"title"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	title, err := normalizeTitle(req.Title)
	if err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	t, err := h.store.Create(r.Context(), title)
	if err != nil {
		internalError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var p Patch
	if err := httpx.Decode(w, r, &p); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if p.Title == nil && p.Completed == nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "nothing to update: send title or completed")
		return
	}
	if p.Title != nil {
		title, err := normalizeTitle(*p.Title)
		if err != nil {
			httpx.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		p.Title = &title
	}

	t, err := h.store.Update(r.Context(), id, p)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	err := h.store.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pathID parses {id}, writing a 400 and returning false when it is invalid.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// internalError logs the real cause and returns a generic 500, so database
// details never leak to clients.
func internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "todo request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	httpx.Error(w, http.StatusInternalServerError, "internal server error")
}

package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/list"
	"github.com/herrnan/handlat/internal/app/recipe/db"
	"github.com/herrnan/handlat/internal/util"
)

const (
	maxNameLength  = 200
	maxIngredients = 100
	maxPortions    = 1<<53 - 1
)

var errDisabled = errors.New("recipes experiment is disabled")

type ingredient struct {
	Name   string     `json:"name"`
	Amount *string    `json:"amount,omitempty"`
	Unit   *list.Unit `json:"unit,omitempty"`
}

type recipe struct {
	ID          string       `json:"id,omitempty"`
	Name        string       `json:"name"`
	Portions    int64        `json:"portions"`
	Ingredients []ingredient `json:"ingredients"`
}

type summary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Handler struct {
	conn    *sql.DB
	queries *db.Queries
	log     *slog.Logger
}

func New(conn *sql.DB, log *slog.Logger) *Handler {
	return &Handler{conn: conn, queries: db.New(conn), log: log}
}

func (h *Handler) Register(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	routes := http.NewServeMux()
	routes.HandleFunc("GET /recipes", h.index)
	routes.HandleFunc("POST /recipes/enable", h.enable)
	routes.HandleFunc("POST /recipes", h.save)
	routes.HandleFunc("GET /recipes/{id}", h.get)
	routes.HandleFunc("PUT /recipes/{id}", h.save)
	routes.HandleFunc("DELETE /recipes/{id}", h.delete)
	protected := requireSession(routes)
	mux.Handle("/recipes", protected)
	mux.Handle("/recipes/", protected)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserID(ctx)

	enabled, err := h.queries.RecipesEnabled(ctx, user)
	if err != nil {
		h.fail(w, err)
		return
	}

	result := struct {
		Enabled bool      `json:"enabled"`
		Recipes []summary `json:"recipes"`
	}{Enabled: enabled == 1, Recipes: []summary{}}

	if result.Enabled {
		rows, err := h.queries.ListRecipes(ctx, user)
		if err != nil {
			h.fail(w, err)
			return
		}

		for _, row := range rows {
			result.Recipes = append(result.Recipes, summary{row.ID, row.Name})
		}
	}

	h.respond(w, http.StatusOK, result)
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	if err := h.queries.EnableRecipes(r.Context(), auth.UserID(r.Context())); err != nil {
		h.fail(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func load(ctx context.Context, q *db.Queries, id string, user []byte) (recipe, error) {
	rows, err := q.GetRecipe(ctx, db.GetRecipeParams{ID: id, UserID: user})
	if err != nil {
		return recipe{}, err
	}

	if len(rows) == 0 {
		return recipe{}, sql.ErrNoRows
	}

	first := rows[0]

	result := recipe{
		ID: first.ID, Name: first.Name, Portions: first.Portions,
		Ingredients: make([]ingredient, 0, len(rows)),
	}
	for _, row := range rows {
		if row.IngredientName != nil {
			result.Ingredients = append(result.Ingredients, ingredient{
				Name: *row.IngredientName, Amount: row.Amount, Unit: (*list.Unit)(row.Unit),
			})
		}
	}

	return result, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	result, err := load(r.Context(), h.queries, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		h.fail(w, err)
		return
	}

	h.respond(w, http.StatusOK, result)
}

func validName(value *string) bool {
	*value = strings.TrimSpace(*value)
	return *value != "" && utf8.RuneCountInString(*value) <= maxNameLength
}

func readRecipe(w http.ResponseWriter, r *http.Request) (recipe, bool) {
	var value recipe

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	decoder.DisallowUnknownFields()

	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF {
		return value, false
	}

	if r.Method == http.MethodPut {
		value.ID = r.PathValue("id")
	}

	id, err := uuid.Parse(value.ID)
	if err != nil || id == uuid.Nil || !validName(&value.Name) ||
		value.Portions < 1 || value.Portions > maxPortions ||
		len(value.Ingredients) == 0 || len(value.Ingredients) > maxIngredients {
		return value, false
	}

	value.ID = id.String()
	for i := range value.Ingredients {
		item := &value.Ingredients[i]
		if !validName(&item.Name) || !list.NormalizeAmount(item.Amount) ||
			(item.Unit != nil && !item.Unit.Valid()) {
			return value, false
		}
	}

	return value, true
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	value, ok := readRecipe(w, r)
	if !ok {
		util.WriteAPIError(w, http.StatusBadRequest, "invalid_recipe", nil)
		return
	}

	created := r.Method == http.MethodPost

	result, err := h.store(r.Context(), value, created)
	if err != nil {
		h.fail(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	h.respond(w, status, result)
}

func (h *Handler) store(ctx context.Context, value recipe, create bool) (recipe, error) {
	tx, err := h.conn.BeginTx(ctx, nil)
	if err != nil {
		return recipe{}, err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			h.log.Error("Roll back recipe transaction", "error", err)
		}
	}()

	q := h.queries.WithTx(tx)

	user := auth.UserID(ctx)
	writeIngredients := true

	if create {
		enabled, err := q.RecipesEnabled(ctx, user)
		if err != nil {
			return recipe{}, err
		}

		if enabled != 1 {
			return recipe{}, errDisabled
		}

		if err := q.EnsureCollection(ctx, user); err != nil {
			return recipe{}, err
		}

		rows, err := q.CreateRecipe(ctx, db.CreateRecipeParams{
			ID: value.ID, UserID: user, Name: value.Name, Portions: value.Portions,
		})
		if err != nil {
			return recipe{}, err
		}

		writeIngredients = rows != 0
	} else {
		rows, err := q.UpdateRecipe(ctx, db.UpdateRecipeParams{
			ID: value.ID, UserID: user, Name: value.Name, Portions: value.Portions,
		})
		if err != nil {
			return recipe{}, err
		}

		if rows == 0 {
			return recipe{}, sql.ErrNoRows
		}

		if err := q.ClearRecipeIngredients(ctx, value.ID); err != nil {
			return recipe{}, err
		}
	}

	if writeIngredients {
		for position, item := range value.Ingredients {
			if err := q.AddRecipeIngredient(ctx, db.AddRecipeIngredientParams{
				RecipeID: value.ID, Position: int64(position), Name: item.Name,
				Amount: item.Amount, Unit: (*string)(item.Unit),
			}); err != nil {
				return recipe{}, err
			}
		}
	}

	result, err := load(ctx, q, value.ID, user)
	if err != nil {
		return recipe{}, err
	}

	return result, tx.Commit()
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.DeleteRecipe(r.Context(), db.DeleteRecipeParams{
		ID: r.PathValue("id"), UserID: auth.UserID(r.Context()),
	})
	if err != nil {
		h.fail(w, err)
		return
	}

	if rows == 0 {
		h.fail(w, sql.ErrNoRows)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.log.Error("Write recipe response", "error", err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		util.WriteAPIError(w, http.StatusNotFound, "recipe_unavailable", nil)
	case errors.Is(err, errDisabled):
		util.WriteAPIError(w, http.StatusForbidden, "recipes_disabled", nil)
	default:
		h.log.Error("Recipe operation failed", "error", err)
		util.WriteAPIError(w, http.StatusInternalServerError, "server_error", nil)
	}
}

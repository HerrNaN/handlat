package list

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/list/db"
)

type Handler struct {
	conn    *sql.DB
	queries *db.Queries
	log     *slog.Logger
	// Serializes subscription snapshots and publication with committed writes.
	mu          sync.Mutex
	subscribers map[string]map[chan listEvent]string
}

type Item struct {
	ID         string     `json:"id"`
	Text       string     `json:"text"`
	Department string     `json:"department"`
	CreatedAt  time.Time  `json:"createdAt"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
}

type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Items       []Item `json:"items"`
	MemberCount int64  `json:"memberCount"`
}

type Summary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ItemCount   int64  `json:"itemCount"`
	MemberCount int64  `json:"memberCount"`
}

func New(conn *sql.DB, log *slog.Logger) *Handler {
	return &Handler{
		conn:        conn,
		queries:     db.New(conn),
		log:         log,
		subscribers: make(map[string]map[chan listEvent]string),
	}
}

func (h *Handler) Register(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	routes := http.NewServeMux()
	routes.HandleFunc("GET /lists", h.index)
	routes.HandleFunc("POST /lists", h.create)
	routes.HandleFunc("GET /lists/{id}", h.get)
	routes.HandleFunc("GET /lists/{id}/events", h.stream)
	routes.HandleFunc("PATCH /lists/{id}", h.rename)
	routes.HandleFunc("DELETE /lists/{id}", h.leaveList)
	routes.HandleFunc("POST /lists/{id}/invitations", h.createInvitation)
	routes.HandleFunc("POST /lists/{id}/items", h.addItem)
	routes.HandleFunc("PATCH /lists/{id}/items/{itemID}", h.updateItem)
	routes.HandleFunc("DELETE /lists/{id}/items/{itemID}", h.deleteItem)
	routes.HandleFunc("DELETE /lists/{id}/checked-items", h.deleteChecked)
	protected := requireSession(routes)
	mux.Handle("/lists", protected)
	mux.Handle("/lists/", protected)
	mux.HandleFunc("GET /invitations/{token}", h.getInvitation)
	mux.Handle(
		"POST /invitations/{token}/accept",
		requireSession(http.HandlerFunc(h.acceptInvitation)),
	)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ListLists(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		h.fail(w, err)
		return
	}

	result := make([]Summary, 0, len(rows))
	for _, row := range rows {
		result = append(result, Summary{row.ID, row.Name, row.ItemCount, row.MemberCount})
	}

	h.respond(w, http.StatusOK, result)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if !validText(body.Name, 200) {
		http.Error(w, "Name must contain 1–200 characters", 400)
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		h.fail(w, err)
		return
	}

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)
	if err := q.CreateList(
		r.Context(),
		db.CreateListParams{ID: id.String(), Name: body.Name, Now: now()},
	); err != nil {
		h.fail(w, err)
		return
	}

	if err := q.AddMember(
		r.Context(),
		db.AddMemberParams{ListID: id.String(), UserID: auth.UserID(r.Context())},
	); err != nil {
		h.fail(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.fail(w, err)
		return
	}

	w.Header().Set("Location", "/api/lists/"+id.String())
	h.respond(
		w,
		http.StatusCreated,
		List{ID: id.String(), Name: body.Name, Items: []Item{}, MemberCount: 1},
	)
}

func snapshot(ctx context.Context, q *db.Queries, value db.List) (List, error) {
	rows, err := q.ListItems(ctx, value.ID)
	if err != nil {
		return List{}, err
	}

	members, err := q.CountMembers(ctx, value.ID)
	if err != nil {
		return List{}, err
	}

	result := List{
		ID:          value.ID,
		Name:        value.Name,
		Items:       make([]Item, 0, len(rows)),
		MemberCount: members,
	}
	for _, row := range rows {
		var checkedAt *time.Time

		if row.CheckedAt != nil {
			timestamp := time.Unix(*row.CheckedAt, 0).UTC()
			checkedAt = &timestamp
		}

		result.Items = append(
			result.Items,
			Item{row.ID, row.Text, row.Department, time.Unix(row.CreatedAt, 0).UTC(), checkedAt},
		)
	}

	return result, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	value, err := accessibleList(r.Context(), q, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		h.fail(w, err)
		return
	}

	result, err := snapshot(r.Context(), q, value)
	if err != nil {
		h.fail(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.fail(w, err)
		return
	}

	h.respond(w, http.StatusOK, result)
}

// Transactions serialize writes in backend order. Authorization, field changes,
// timestamps, and the returned snapshot are atomic; no client version is needed.
func (h *Handler) commitChange(r *http.Request, change func(*db.Queries) error) (List, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		return List{}, err
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	value, err := accessibleList(r.Context(), q, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		return List{}, err
	}

	if err := q.TouchList(r.Context(), db.TouchListParams{ID: value.ID, Now: now()}); err != nil {
		return List{}, err
	}

	if err := change(q); err != nil {
		return List{}, err
	}

	value, err = q.GetList(r.Context(), value.ID)
	if err != nil {
		return List{}, err
	}

	result, err := snapshot(r.Context(), q, value)
	if err != nil {
		return List{}, err
	}

	if err := tx.Commit(); err != nil {
		return List{}, err
	}

	h.publish(value.ID, listEvent{List: result})

	return result, nil
}

func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, change func(*db.Queries) error) {
	result, err := h.commitChange(r, change)
	if err != nil {
		h.fail(w, err)
		return
	}

	h.respond(w, http.StatusOK, result)
}

func (h *Handler) commitLeave(r *http.Request) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	value, err := accessibleList(r.Context(), q, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		return err
	}

	if err := q.RemoveMember(
		r.Context(),
		db.RemoveMemberParams{ListID: value.ID, UserID: auth.UserID(r.Context())},
	); err != nil {
		return err
	}

	if err := q.DeleteEmptyList(r.Context(), value.ID); err != nil {
		return err
	}

	members, err := q.CountMembers(r.Context(), value.ID)
	if err != nil {
		return err
	}

	var remaining List
	if members > 0 {
		remaining, err = snapshot(r.Context(), q, value)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	h.revoke(value.ID, auth.UserID(r.Context()))

	if members > 0 {
		h.publish(value.ID, listEvent{List: remaining})
	}

	return nil
}

func (h *Handler) leaveList(w http.ResponseWriter, r *http.Request) {
	if err := h.commitLeave(r); err != nil {
		h.fail(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if !validText(body.Name, 200) {
		http.Error(w, "Name must contain 1–200 characters", 400)
		return
	}

	h.mutate(w, r, func(q *db.Queries) error {
		return q.RenameList(
			r.Context(),
			db.RenameListParams{ID: r.PathValue("id"), Name: body.Name},
		)
	})
}

type itemInput struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Department string `json:"department"`
}

func readItem(w http.ResponseWriter, r *http.Request) (itemInput, bool) {
	var body itemInput
	if !decode(w, r, &body) {
		return body, false
	}

	body.Text = strings.TrimSpace(body.Text)
	if body.Department == "" {
		body.Department = "other"
	}

	id, err := uuid.Parse(body.ID)
	if err != nil || id == uuid.Nil {
		http.Error(w, "A non-zero item UUID is required", http.StatusBadRequest)
		return body, false
	}

	body.ID = id.String()
	if !validText(body.Text, 2000) {
		http.Error(w, "Text must contain 1–2000 characters", http.StatusBadRequest)
		return body, false
	}

	if !validText(body.Department, 100) {
		http.Error(w, "Department must contain 1–100 characters", http.StatusBadRequest)
		return body, false
	}

	return body, true
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	body, ok := readItem(w, r)
	if !ok {
		return
	}

	h.mutate(w, r, func(q *db.Queries) error {
		return q.AddItem(
			r.Context(),
			db.AddItemParams{
				ID:         body.ID,
				ListID:     r.PathValue("id"),
				Text:       body.Text,
				Department: body.Department,
				CreatedAt:  now(),
			},
		)
	})
}

// A field is either absent or carries a value; null is not a command to clear it.
// Unchecking is represented by checked:false, not a browser-authored timestamp.
type optional[T any] struct {
	set   bool
	value T
}

func (field *optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("field cannot be null")
	}

	field.set = true

	return json.Unmarshal(data, &field.value)
}

func (field optional[T]) pointer() *T {
	if !field.set {
		return nil
	}

	return &field.value
}

func (h *Handler) updateItem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text       optional[string] `json:"text"`
		Department optional[string] `json:"department"`
		Checked    optional[bool]   `json:"checked"`
	}
	if !decode(w, r, &body) {
		return
	}

	if body.Text.set {
		body.Text.value = strings.TrimSpace(body.Text.value)
	}

	if !body.Text.set && !body.Department.set && !body.Checked.set {
		http.Error(w, "Provide text, department or checked fields", http.StatusBadRequest)
		return
	}

	if body.Text.set && !validText(body.Text.value, 2000) {
		http.Error(w, "Text must contain 1–2000 characters", http.StatusBadRequest)
		return
	}

	if body.Department.set && !validText(body.Department.value, 100) {
		http.Error(w, "Department must contain 1–100 characters", http.StatusBadRequest)
		return
	}

	h.mutate(w, r, func(q *db.Queries) error {
		var setChecked, checked int64
		if body.Checked.set {
			setChecked = 1
		}

		if body.Checked.value {
			checked = 1
		}

		return q.UpdateItem(
			r.Context(),
			db.UpdateItemParams{
				ID:         r.PathValue("itemID"),
				ListID:     r.PathValue("id"),
				Text:       body.Text.pointer(),
				Department: body.Department.pointer(),
				SetChecked: setChecked,
				Checked:    checked,
				Now:        now(),
			},
		)
	})
}

func (h *Handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(q *db.Queries) error {
		return q.DeleteItem(
			r.Context(),
			db.DeleteItemParams{ID: r.PathValue("itemID"), ListID: r.PathValue("id"), Now: now()},
		)
	})
}

func (h *Handler) deleteChecked(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(q *db.Queries) error {
		return q.DeleteCheckedItems(
			r.Context(),
			db.DeleteCheckedItemsParams{ListID: r.PathValue("id"), Now: now()},
		)
	})
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return false
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "Expected one JSON value", 400)
		return false
	}

	return true
}

func validText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= max
}
func now() int64 { return time.Now().Unix() }

func (h *Handler) respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.log.Error("Write list response", "error", err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "List or item not found", http.StatusNotFound)
		return
	}

	h.log.Error("List operation failed", "error", err)
	http.Error(w, "Server error", http.StatusInternalServerError)
}

func (h *Handler) rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		h.log.Error("Roll back list transaction", "error", err)
	}
}

package list

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/herrnan/handlat/internal/app/auth"
)

type listEvent struct {
	List    List
	Deleted bool
}

// publish runs under mu, after commit. Keep only the latest snapshot for a slow
// subscriber: full snapshots supersede intermediate states and never block writes.
func (h *Handler) publish(id string, event listEvent) {
	for events := range h.subscribers[id] {
		select {
		case <-events:
		default:
		}

		events <- event
	}
}

// Stop all of the departing user's streams before publishing further changes.
// Called under mu, after the membership removal commits.
func (h *Handler) revoke(id string, userID []byte) {
	for events, member := range h.subscribers[id] {
		if member != string(userID) {
			continue
		}

		select {
		case <-events:
		default:
		}

		events <- listEvent{Deleted: true}

		delete(h.subscribers[id], events)
	}
}

func (h *Handler) subscribe(r *http.Request) (List, chan listEvent, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		return List{}, nil, err
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	value, err := accessibleList(r.Context(), q, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		return List{}, nil, err
	}

	initial, err := snapshot(r.Context(), q, value)
	if err != nil {
		return List{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return List{}, nil, err
	}

	events := make(chan listEvent, 1)
	if h.subscribers[value.ID] == nil {
		h.subscribers[value.ID] = make(map[chan listEvent]string)
	}

	h.subscribers[value.ID][events] = string(auth.UserID(r.Context()))

	return initial, events, nil
}

func (h *Handler) unsubscribe(id string, events chan listEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.subscribers[id], events)

	if len(h.subscribers[id]) == 0 {
		delete(h.subscribers, id)
	}
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	initial, events, err := h.subscribe(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer h.unsubscribe(initial.ID, events)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)

	send := func(event listEvent) error {
		// Bound individual writes without imposing an idle timeout on the stream.
		if err := controller.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return err
		}
		defer func() { _ = controller.SetWriteDeadline(time.Time{}) }()

		if event.Deleted {
			if _, err := fmt.Fprint(w, "event: deleted\ndata: {}\n\n"); err != nil {
				return err
			}
		} else {
			data, err := json.Marshal(event.List)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return err
			}
		}

		return controller.Flush()
	}
	if err := send(listEvent{List: initial}); err != nil {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			if err := send(event); err != nil || event.Deleted {
				return
			}
		}
	}
}

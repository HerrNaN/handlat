package list

import (
	"bytes"
	"context"
	"database/sql"

	"github.com/herrnan/handlat/internal/app/list/db"
)

// accessibleList is the single authorization point for list operations.
// Return the same result for missing and inaccessible lists to avoid leaking IDs.
// Membership permissions can extend this policy when sharing is introduced.
func accessibleList(ctx context.Context, q *db.Queries, id string, userID []byte) (db.List, error) {
	value, err := q.GetList(ctx, id)
	if err != nil {
		return db.List{}, err
	}
	if len(userID) == 0 || !bytes.Equal(value.OwnerID, userID) {
		return db.List{}, sql.ErrNoRows
	}
	return value, nil
}

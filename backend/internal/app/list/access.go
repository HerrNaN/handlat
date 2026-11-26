package list

import (
	"context"

	"github.com/herrnan/handlat/internal/app/list/db"
)

// All list operations use membership for authorization. Missing and inaccessible
// lists return the same result to avoid leaking IDs.
func accessibleList(ctx context.Context, q *db.Queries, id string, userID []byte) (db.List, error) {
	return q.GetMemberList(ctx, db.GetMemberListParams{ID: id, UserID: userID})
}

func addMember(ctx context.Context, q *db.Queries, id string, userID []byte) error {
	if err := q.AddMember(ctx, db.AddMemberParams{ListID: id, UserID: userID}); err != nil {
		return err
	}

	return q.EnsureDefaultShoppingList(ctx, userID)
}

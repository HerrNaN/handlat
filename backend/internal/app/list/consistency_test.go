package list

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

func TestPersistenceAcrossDatabaseReopen(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Persistent"}`), 201)
	path := "/lists/" + list.ID
	body := addition(uuid.NewString(), "Deleted milk")
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, body), 200)
	list = readList(t, request(f.handler, "DELETE", path+"/items/"+list.Items[0].ID, f.owner, ""), 200)
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), "Bread")), 200)
	var sequence int
	var name, filename string
	require.NoError(t, f.conn.QueryRowContext(t.Context(), "PRAGMA database_list").Scan(&sequence, &name, &filename))
	require.NoError(t, f.conn.Close())
	reopened, err := database.Open(t.Context(), filename)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reopened.Close()) })
	require.NoError(t, migrations.Up(reopened))
	handler := apiHandler(t, reopened)
	fetched := readList(t, request(handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, list, fetched)
	retried := readList(t, request(handler, "POST", path+"/items", f.owner, body), 200)
	require.Equal(t, list.Items, retried.Items, "tombstones must survive restarts too")
}

func concurrent(t *testing.T, f fixture, path string, operations ...struct{ method, suffix, body string }) []List {
	t.Helper()
	results := make(chan *httptest.ResponseRecorder, len(operations))
	var wg sync.WaitGroup
	for _, operation := range operations {
		wg.Go(func() {
			results <- request(f.handler, operation.method, path+operation.suffix, f.owner, operation.body)
		})
	}
	wg.Wait()
	close(results)
	values := make([]List, 0, len(operations))
	for result := range results {
		value := readList(t, result, 200)
		values = append(values, value)
	}
	return values
}

func TestConcurrentAdditionsAndRetries(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "distinct additions", true: "same addition retried"}[retry], func(t *testing.T) {
			f := newFixture(t)
			list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Concurrent"}`), 201)
			path := "/lists/" + list.ID
			first, second := uuid.NewString(), uuid.NewString()
			if retry {
				second = first
			}
			concurrent(
				t, f, path,
				struct{ method, suffix, body string }{"POST", "/items", addition(first, "Milk")},
				struct{ method, suffix, body string }{"POST", "/items", addition(second, "Milk")},
			)
			final := readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
			count := 2
			if retry {
				count = 1
			}
			require.Len(t, final.Items, count)
		})
	}
}

func TestConcurrentIndependentFieldsAndSameFieldOrdering(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Concurrent"}`), 201)
	path := "/lists/" + list.ID
	id := uuid.NewString()
	_ = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(id, "Milk")), 200)
	suffix := "/items/" + id
	concurrent(
		t, f, path,
		struct{ method, suffix, body string }{"PATCH", suffix, `{"text":"Oat milk"}`},
		struct{ method, suffix, body string }{"PATCH", suffix, `{"department":"dairy"}`},
		struct{ method, suffix, body string }{"PATCH", suffix, `{"checked":true}`},
	)
	final := readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, "Oat milk", final.Items[0].Text)
	require.Equal(t, "dairy", final.Items[0].Department)
	require.NotNil(t, final.Items[0].CheckedAt)
	results := concurrent(
		t, f, path,
		struct{ method, suffix, body string }{"PATCH", suffix, `{"text":"First client"}`},
		struct{ method, suffix, body string }{"PATCH", suffix, `{"text":"Second client"}`},
	)
	final = readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	// Completion order is not processing order. The final state must match one
	// atomic response, and one response must have observed each client's value.
	require.ElementsMatch(t, []string{"First client", "Second client"}, []string{results[0].Items[0].Text, results[1].Items[0].Text})
	require.Contains(t, results, final)

	// A later write deterministically wins without exposing an ordering token.
	last := readList(t, request(f.handler, "PATCH", path+suffix, f.owner, `{"text":"Last write"}`), 200)
	final = readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, last, final)
	concurrent(
		t, f, path,
		struct{ method, suffix, body string }{"DELETE", suffix, ""},
		struct{ method, suffix, body string }{"PATCH", suffix, `{"text":"Concurrent edit","checked":false}`},
	)
	final = readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Empty(t, final.Items, "deletion wins regardless of processing order")
}

func TestFailedWriteRollsBackChanges(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Rollback"}`), 201)
	path := "/lists/" + list.ID
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), "Milk")), 200)
	_, err := f.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_item_update BEFORE UPDATE ON items BEGIN SELECT RAISE(ABORT, 'test write failure'); END`)
	require.NoError(t, err)
	result := request(f.handler, "PATCH", path+"/items/"+list.Items[0].ID, f.owner, `{"text":"Not saved"}`)
	require.Equal(t, 500, result.Code)
	fetched := readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, list, fetched)
}

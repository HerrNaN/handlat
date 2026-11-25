package list_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	listapi "github.com/herrnan/handlat/internal/app/list"
	listdb "github.com/herrnan/handlat/internal/app/list/db"
	"github.com/stretchr/testify/require"
)

func TestAddItemMergesCompatibleQuantities(t *testing.T) {
	tests := []struct {
		name                   string
		firstAmount, firstUnit string
		addAmount, addUnit     string
		wantAmount, wantUnit   string
	}{
		{"weight", "1", "kg", "500", "g", "1.5", "kg"},
		{"volume", "1", "l", "150", "ml", "1.15", "l"},
		{"nonterminating in the original unit", "1", "msk", "1", "tsk", "20", "ml"},
		{"unitless counts", "2", "", "3", "", "5", ""},
		{"package counts", "2", "pkt", "3", "pkt", "5", "pkt"},
		{
			"exact decimal",
			"9007199254740993.123456789",
			"g",
			"0.000000001",
			"g",
			"9007199254740993.12345679",
			"g",
		},
		{"different dimensions", "1", "g", "1", "ml", "", ""},
		{"different count units", "1", "st", "1", "pkt", "", ""},
		{"only one unit", "2", "", "3", "st", "", ""},
		{"only one amount", "", "", "1", "l", "", ""},
		{"total exceeds quantity limit", strings.Repeat("9", 100), "g", "1", "g", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newMergeFixture(t)
			first := uuid.NewString()
			incoming := uuid.NewString()

			created := addAndRead(
				t,
				f,
				additionRequest(t, f, first, "Sea  salt", tc.firstAmount, tc.firstUnit),
			)
			require.Empty(t, created.MergedItemID)

			result := addAndRead(
				t,
				f,
				additionRequest(t, f, incoming, " sea SALT ", tc.addAmount, tc.addUnit),
			)

			if tc.wantAmount == "" {
				require.Len(t, result.Items, 2)
				require.NotEqual(t, result.Items[0].ID, result.Items[1].ID)
				require.Empty(t, result.MergedItemID)

				return
			}

			require.Len(t, result.Items, 1)
			require.Equal(t, first, result.Items[0].ID)
			require.Equal(t, first, result.MergedItemID)
			require.Equal(t, "Sea  salt", result.Items[0].Text)
			require.Equal(t, tc.wantAmount, *result.Items[0].Amount)

			if tc.wantUnit == "" {
				require.Nil(t, result.Items[0].Unit)
			} else {
				require.Equal(t, tc.wantUnit, string(*result.Items[0].Unit))
			}
		})
	}
}

func TestAddItemMergesIntoFirstEligibleRow(t *testing.T) {
	f := newMergeFixture(t)
	q := listdb.New(f.conn)
	ctx := t.Context()

	// The two compatible rows have the same creation time, but are inserted in reverse ID order.
	for _, row := range []listdb.AddItemParams{
		{ID: "deleted", CreatedAt: 1, Amount: new("1"), Unit: new("kg"), DeletedAt: new(int64(1))},
		{ID: "checked", CreatedAt: 2, Amount: new("1"), Unit: new("kg")},
		{ID: "unmeasured", CreatedAt: 3},
		{ID: "package", CreatedAt: 4, Amount: new("1"), Unit: new("pkt")},
		{ID: "b", CreatedAt: 5, Amount: new("250"), Unit: new("g")},
		{ID: "a", CreatedAt: 5, Amount: new("1"), Unit: new("kg")},
	} {
		row.ListID, row.Text, row.Department = "list", "Potatoes", "produce"
		require.NoError(t, q.AddItem(ctx, row))
	}

	require.NoError(t, q.UpdateItem(ctx, listdb.UpdateItemParams{
		ListID: "list", ID: "checked", SetChecked: 1, Checked: 1, Now: 2,
	}))

	incoming := uuid.NewString()
	result := addAndRead(t, f, additionRequest(t, f, incoming, "POTATOES", "500", "g"))
	require.Len(t, result.Items, 5)
	require.NotNil(t, result.Items[0].CheckedAt)
	require.Equal(t, "1", *result.Items[0].Amount)

	target := result.Items[3]
	require.Equal(t, "a", target.ID)
	require.Equal(t, "Potatoes", target.Text)
	require.Equal(t, "produce", target.Department)
	require.EqualValues(t, 5, target.CreatedAt.Unix())
	require.Equal(t, "1.5", *target.Amount)
	require.Equal(t, listapi.UnitKilogram, *target.Unit)
	require.Equal(t, "250", *result.Items[4].Amount)

	retry := addAndRead(t, f, additionRequest(t, f, incoming, "Potatoes", "500", "g"))
	require.Equal(t, result.List, retry.List)
	require.Empty(t, retry.MergedItemID)
	require.NoError(
		t,
		q.DeleteItem(ctx, listdb.DeleteItemParams{ListID: "list", ID: target.ID, Now: 6}),
	)

	afterDelete := addAndRead(t, f, additionRequest(t, f, incoming, "Potatoes", "500", "g"))
	require.Len(t, afterDelete.Items, 4)
	require.Equal(t, "b", afterDelete.Items[3].ID)
	require.Equal(t, "250", *afterDelete.Items[3].Amount)
}

func TestConcurrentItemAddsCountEachRequestIDOnce(t *testing.T) {
	f := newMergeFixture(t)
	repeatedID := uuid.NewString()
	ids := []string{repeatedID, uuid.NewString(), repeatedID, uuid.NewString()}
	responses := make([]*httptest.ResponseRecorder, len(ids))
	start := make(chan struct{})

	var requests sync.WaitGroup

	for i, id := range ids {
		req := additionRequest(t, f, id, "Apples", "1", "")
		responses[i] = httptest.NewRecorder()

		requests.Go(func() {
			<-start
			f.handler.ServeHTTP(responses[i], req)
		})
	}

	close(start)
	requests.Wait()

	for _, response := range responses {
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}

	result := addAndRead(t, f, additionRequest(t, f, repeatedID, "Apples", "1", ""))
	require.Len(t, result.Items, 1)
	require.Equal(t, "3", *result.Items[0].Amount)
}

func newMergeFixture(t *testing.T) fixture {
	t.Helper()
	f := newFixture(t)
	require.NoError(t, seedList(t.Context(), listdb.New(f.conn), seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: 1,
	}))

	return f
}

func additionRequest(t *testing.T, f fixture, id, name, amount, unit string) *http.Request {
	t.Helper()

	payload, err := json.Marshal(struct {
		ID     string `json:"id"`
		Text   string `json:"text"`
		Amount string `json:"amount,omitempty"`
		Unit   string `json:"unit,omitempty"`
	}{id, name, amount, unit})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/lists/list/items",
		bytes.NewReader(payload),
	)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	return req
}

type additionResponse struct {
	listapi.List
	MergedItemID string `json:"mergedItemId"`
}

func addAndRead(t *testing.T, f fixture, req *http.Request) additionResponse {
	t.Helper()

	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var result additionResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))

	return result
}

func TestBatchAdditionIsAtomicAndRetryable(t *testing.T) {
	f := newMergeFixture(t)
	first := uuid.NewString()
	addAndRead(t, f, additionRequest(t, f, first, "Tomatoes", "100", "g"))

	itemID := uuid.NewString()
	batch := `[{"id":"` + itemID + `","text":"Tomatoes","amount":"200","unit":"g"},` +
		`{"id":"` + uuid.NewString() + `","text":"Basil"}]`
	request := func(body string) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(
			http.MethodPost,
			"/lists/list/items/batch",
			strings.NewReader(body),
		)
		req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, req)

		return response
	}

	invalid := request(strings.Replace(batch, "Basil", "", 1))
	require.Equal(t, http.StatusBadRequest, invalid.Code, invalid.Body.String())
	current := addAndRead(t, f, additionRequest(t, f, first, "Tomatoes", "100", "g"))
	require.Len(t, current.Items, 1)
	require.Equal(t, "100", *current.Items[0].Amount)

	response := request(batch)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var result listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Len(t, result.Items, 2)

	for _, item := range result.Items {
		if item.ID == first {
			require.Equal(t, "300", *item.Amount)
		} else {
			require.Equal(t, "Basil", item.Text)
			require.Nil(t, item.Amount)
		}
	}

	retry := request(batch)
	require.Equal(t, http.StatusOK, retry.Code, retry.Body.String())
	require.JSONEq(t, response.Body.String(), retry.Body.String())
}

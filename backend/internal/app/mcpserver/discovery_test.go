package mcpserver_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mismatchedMetadata = `{"client_id":"https://other.example/client.json","redirect_uris":["` + redirectURI + `"]}`

func TestClientIDMismatchInvalidatesOnlyMatchingRefreshTokens(t *testing.T) {
	f := newFixture(t)
	issued := f.decodeTokens(f.token(f.server, codeForm(f.approve())))
	require.NotEmpty(t, issued.Refresh)

	otherClient := "https://other.example/client.json"

	otherResource := "https://other.example/mcp"
	for _, unrelated := range []struct {
		grantID  string
		user     []byte
		client   string
		resource string
	}{
		{"other-client", f.user, otherClient, resource},
		{"other-resource", []byte("other"), clientID, otherResource},
	} {
		f.exec(
			`
			INSERT INTO mcp_grants (id, user_id, client_id, client_name, scope, resource, created_at)
			VALUES (?, ?, ?, 'Other', ?, ?, unixepoch())`,
			unrelated.grantID, unrelated.user, unrelated.client, scope, unrelated.resource,
		)
		hash := sha256.Sum256([]byte(unrelated.grantID))
		f.exec(
			"INSERT INTO mcp_refresh_tokens (token_hash, grant_id) VALUES (?, ?)",
			hash[:],
			unrelated.grantID,
		)
	}

	f.useMetadata(mismatchedMetadata)
	response := f.authorize(authorizationQuery(redirectURI))
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
	assert.Empty(t, response.Header().Get("Location"))

	rejected := f.token(f.server, refreshForm(issued.Refresh))
	assert.Equal(t, http.StatusBadRequest, rejected.Code)
	assert.JSONEq(t, `{"error":"invalid_grant"}`, rejected.Body.String())
	assert.Equal(t, http.StatusOK, f.rpc(f.server, issued.Access, "tools/list", "{}").Code)

	for _, untargeted := range []string{"other-client", "other-resource"} {
		var count int

		err := f.db.QueryRowContext(
			t.Context(),
			"SELECT COUNT(*) FROM mcp_refresh_tokens WHERE grant_id = ?", untargeted,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	}
}

func TestClientIDMismatchCancelsPendingCredentials(t *testing.T) {
	f := newFixture(t)
	code := f.approve()
	path, browser := f.beginApproval(redirectURI)

	f.useMetadata(mismatchedMetadata)
	response := f.authorize(authorizationQuery(redirectURI))
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())

	assert.Equal(t, http.StatusNotFound, f.approvalRequest(http.MethodGet, path, browser, "").Code)
	assert.JSONEq(t, `{"error":"invalid_grant"}`, f.token(f.server, codeForm(code)).Body.String())
}

func TestMetadataFetchFailuresPreserveRenewal(t *testing.T) {
	f := newFixture(t)

	issued := f.decodeTokens(f.token(f.server, codeForm(f.approve())))
	for _, fail := range []metadataTransport{
		func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("metadata transport unavailable")
		},
		func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}, nil
		},
	} {
		f.useMetadataTransport(fail)
		response := f.authorize(authorizationQuery(redirectURI))
		assert.Equal(t, http.StatusServiceUnavailable, response.Code)
		assert.JSONEq(t, `{"error":"temporarily_unavailable"}`, response.Body.String())
		assert.Empty(t, response.Header().Get("Location"))

		issued = f.decodeTokens(f.token(f.server, refreshForm(issued.Refresh)))
		assert.NotEmpty(t, issued.Refresh)
	}

	f.useMetadata(`{"client_name":"Missing client ID"}`)
	response := f.authorize(authorizationQuery(redirectURI))
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
	assert.NotEmpty(t, f.decodeTokens(f.token(f.server, refreshForm(issued.Refresh))).Refresh)
}

func TestMetadataFetchesAreBoundedAndCoalesced(t *testing.T) {
	f := newFixture(t)

	synctest.Test(t, func(t *testing.T) {
		const limit = 8

		started := make(chan string, limit+3)
		release := make(chan struct{})

		free := sync.OnceFunc(func() { close(release) })
		defer free()

		f.useMetadataTransport(
			metadataTransport(func(request *http.Request) (*http.Response, error) {
				started <- request.URL.String()

				<-release

				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}, nil
			}),
		)

		authorizeClient := func(id string) *httptest.ResponseRecorder {
			query := authorizationQuery(redirectURI)
			query.Set("client_id", id)

			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, origin+"/oauth/authorize?"+query.Encode(), nil,
			)

			return perform(f.server, request)
		}

		replies := make(chan *httptest.ResponseRecorder, limit+1)
		for i := range limit {
			go func() { replies <- authorizeClient(fmt.Sprintf("%s?lookup=%d", clientID, i)) }()
		}

		synctest.Wait()
		require.Len(t, started, limit)

		saturated := authorizeClient(clientID + "?lookup=extra")
		assert.Equal(t, http.StatusServiceUnavailable, saturated.Code)
		assert.Equal(t, "1", saturated.Header().Get("Retry-After"))
		assert.Empty(t, saturated.Header().Get("Location"))
		assert.JSONEq(t, `{"error":"temporarily_unavailable"}`, saturated.Body.String())

		go func() { replies <- authorizeClient(clientID + "?lookup=0") }()

		synctest.Wait()
		assert.Empty(t, replies)
		assert.Len(t, started, limit)

		free()

		for range limit + 1 {
			response := <-replies
			assert.Equal(t, http.StatusServiceUnavailable, response.Code)
			assert.Empty(t, response.Header().Get("Retry-After"))
		}

		retried := authorizeClient(clientID + "?lookup=extra")
		assert.Empty(t, retried.Header().Get("Retry-After"))
		assert.Len(t, started, limit+1)
	})
}

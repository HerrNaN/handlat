package mcpserver_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/list"
	"github.com/herrnan/handlat/internal/app/mcpserver"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/herrnan/handlat/internal/app/recipe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	origin      = "https://handlat.example"
	resource    = origin + "/mcp"
	clientID    = "https://client.example/oauth/client.json"
	redirectURI = "https://callback.example/oauth/callback?original=1"
	verifier    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	scope       = "handlat:lists-recipes"
)

const metadataIdentity = `"client_id":"` + clientID + `","client_name":"Example agent","redirect_uris":["` + redirectURI + `"]`

const defaultMetadata = `{` + metadataIdentity + `,"grant_types":["authorization_code","refresh_token"],"response_types":["code"]}`

type fixture struct {
	t        *testing.T
	db       *sql.DB
	user     []byte
	auth     *auth.Handler
	key      *ecdsa.PrivateKey
	count    uint32
	server   http.Handler
	metadata http.RoundTripper
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	db, err := database.Open(t.Context(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })
	require.NoError(t, migrations.Up(db))

	logger := slog.New(slog.DiscardHandler)
	authHandler, err := auth.New(db, "handlat.example", origin, logger)
	require.NoError(t, err)

	f := &fixture{t: t, db: db, user: []byte("owner"), auth: authHandler}
	f.exec(
		"INSERT INTO users (id, name) VALUES (?, 'Owner'), (?, 'Other')",
		f.user,
		[]byte("other"),
	)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f.key = key
	publicKey, err := cbor.Marshal(map[int]any{
		1: 2, 3: -7, -1: 1,
		-2: key.X.FillBytes(make([]byte, 32)),
		-3: key.Y.FillBytes(make([]byte, 32)),
	})
	require.NoError(t, err)
	credential, err := json.Marshal(
		webauthn.Credential{ID: []byte("credential"), PublicKey: publicKey},
	)
	require.NoError(t, err)
	f.exec(
		"INSERT INTO credentials (id, user_id, data) VALUES (?, ?, ?)",
		[]byte("credential"),
		f.user,
		string(credential),
	)
	f.useMetadata(defaultMetadata)

	return f
}

func (f *fixture) handler(issuer string, enabled bool) http.Handler {
	root, browser, operations := http.NewServeMux(), http.NewServeMux(), http.NewServeMux()
	identity := func(next http.Handler) http.Handler { return next }
	list.New(f.db, slog.New(slog.DiscardHandler)).Register(operations, identity)
	recipe.New(f.db, slog.New(slog.DiscardHandler)).Register(operations, identity)

	handler := mcpserver.New(f.db, slog.New(slog.DiscardHandler), f.auth, issuer, enabled)
	if f.metadata != nil {
		handler.SetMetadataTransportForTest(f.metadata)
	}

	handler.Register(root, browser, operations, "test")
	root.Handle("/api/", http.NewCrossOriginProtection().Handler(http.StripPrefix("/api", browser)))

	return root
}

type metadataTransport func(*http.Request) (*http.Response, error)

func (transport metadataTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func (f *fixture) useMetadata(metadata string) {
	if metadata == "" {
		f.useMetadataTransport(nil)
		return
	}

	f.useMetadataTransport(metadataTransport(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}))
}

func (f *fixture) useMetadataTransport(transport http.RoundTripper) {
	f.metadata = transport
	f.server = f.handler(origin, true)
}

func (f *fixture) exec(query string, args ...any) {
	f.t.Helper()

	_, err := f.db.ExecContext(f.t.Context(), query, args...)
	require.NoError(f.t, err)
}

func perform(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func authorizationQuery(redirect string) url.Values {
	challenge := sha256.Sum256([]byte(verifier))

	return url.Values{
		"response_type": {"code"}, "client_id": {clientID},
		"redirect_uri": {redirect}, "resource": {resource}, "scope": {scope},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"}, "state": {"client-state"},
	}
}

func (f *fixture) authorize(query url.Values) *httptest.ResponseRecorder {
	f.t.Helper()

	request := httptest.NewRequestWithContext(
		f.t.Context(), http.MethodGet, origin+"/oauth/authorize?"+query.Encode(), nil,
	)

	return perform(f.server, request)
}

func (f *fixture) beginApproval(redirect string) (string, string) {
	f.t.Helper()

	response := f.authorize(authorizationQuery(redirect))
	require.Equal(f.t, http.StatusSeeOther, response.Code, response.Body.String())
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(f.t, err)
	require.Equal(f.t, "/connect", location.Path)
	require.Empty(f.t, location.Host)
	id := location.Query().Get("request")
	require.NotEmpty(f.t, id)

	result := response.Result()

	f.t.Cleanup(func() { assert.NoError(f.t, result.Body.Close()) })

	cookies := result.Cookies()
	require.Len(f.t, cookies, 1)
	require.Equal(f.t, "handlat_mcp_approval", cookies[0].Name)

	return "/api/mcp/approvals/" + id, cookies[0].Value
}

func (f *fixture) approvalRequest(
	method, path, browser, body string,
	cookies ...*http.Cookie,
) *httptest.ResponseRecorder {
	f.t.Helper()

	request := httptest.NewRequestWithContext(
		f.t.Context(),
		method,
		origin+path,
		strings.NewReader(body),
	)
	request.AddCookie(&http.Cookie{Name: "handlat_mcp_approval", Value: browser})

	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	return perform(f.server, request)
}

func (f *fixture) assertion(challenge string) string {
	f.t.Helper()

	clientData, err := json.Marshal(map[string]string{
		"type": "webauthn.get", "challenge": challenge, "origin": origin,
	})
	require.NoError(f.t, err)

	rpHash := sha256.Sum256([]byte("handlat.example"))
	data := append(rpHash[:], 0x05) // User present and verified.
	f.count++
	data = binary.BigEndian.AppendUint32(data, f.count)
	clientHash := sha256.Sum256(clientData)
	message := sha256.Sum256(append(append([]byte{}, data...), clientHash[:]...))
	signature, err := ecdsa.SignASN1(rand.Reader, f.key, message[:])
	require.NoError(f.t, err)

	encode := base64.RawURLEncoding.EncodeToString
	credential := encode([]byte("credential"))
	body, err := json.Marshal(map[string]any{
		"id": credential, "rawId": credential, "type": "public-key",
		"response": map[string]string{
			"clientDataJSON": encode(clientData), "authenticatorData": encode(data),
			"signature": encode(signature), "userHandle": encode(f.user),
		},
	})
	require.NoError(f.t, err)

	return string(body)
}

func (f *fixture) approve() string {
	f.t.Helper()

	root, browser := f.beginApproval(redirectURI)
	require.Equal(
		f.t,
		http.StatusNotFound,
		f.approvalRequest(http.MethodGet, root, "wrong-browser", "").Code,
	)
	info := f.approvalRequest(http.MethodGet, root, browser, "")
	require.Equal(f.t, http.StatusOK, info.Code)
	require.Equal(
		f.t, http.StatusBadRequest,
		f.approvalRequest(http.MethodPost, root+"/passkey/finish", browser, "{}").Code,
	)

	begin := f.approvalRequest(http.MethodPost, root+"/passkey/begin", browser, "")
	require.Equal(f.t, http.StatusOK, begin.Code)

	var options struct {
		Challenge string `json:"challenge"`
	}
	require.NoError(f.t, json.Unmarshal(begin.Body.Bytes(), &options))
	require.NotEmpty(f.t, options.Challenge)

	result := begin.Result()

	f.t.Cleanup(func() { assert.NoError(f.t, result.Body.Close()) })

	cookies := result.Cookies()
	require.Len(f.t, cookies, 1)

	finish := f.approvalRequest(
		http.MethodPost,
		root+"/passkey/finish",
		browser,
		f.assertion(options.Challenge),
		cookies[0],
	)
	require.Equal(f.t, http.StatusOK, finish.Code, finish.Body.String())

	var response struct {
		Redirect string `json:"redirect"`
	}
	require.NoError(f.t, json.Unmarshal(finish.Body.Bytes(), &response))
	callback, err := url.Parse(response.Redirect)
	require.NoError(f.t, err)
	assert.Equal(f.t, "callback.example", callback.Host)
	assert.Equal(f.t, "1", callback.Query().Get("original"))
	assert.Equal(f.t, "client-state", callback.Query().Get("state"))
	assert.Equal(f.t, origin, callback.Query().Get("iss"))
	require.NotEmpty(f.t, callback.Query().Get("code"))

	return callback.Query().Get("code")
}

func (f *fixture) token(server http.Handler, form url.Values) *httptest.ResponseRecorder {
	f.t.Helper()

	req := httptest.NewRequestWithContext(
		f.t.Context(),
		http.MethodPost,
		origin+"/oauth/token",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return perform(server, req)
}

type tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

func (f *fixture) decodeTokens(response *httptest.ResponseRecorder) tokens {
	f.t.Helper()

	require.Equal(f.t, http.StatusOK, response.Code, response.Body.String())

	var result tokens
	require.NoError(f.t, json.Unmarshal(response.Body.Bytes(), &result))
	require.NotEmpty(f.t, result.Access)

	return result
}

func codeForm(code string) url.Values {
	return url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID},
		"redirect_uri": {redirectURI}, "code": {code}, "code_verifier": {verifier},
		"resource": {resource},
	}
}

func refreshForm(token string) url.Values {
	return url.Values{
		"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {token},
	}
}

func (f *fixture) rpc(
	server http.Handler,
	token, method, params string,
) *httptest.ResponseRecorder {
	f.t.Helper()

	body := `{"jsonrpc":"2.0","method":"` + method + `","id":1,"params":` + params + `}`
	req := httptest.NewRequestWithContext(
		f.t.Context(),
		http.MethodPost,
		resource,
		strings.NewReader(body),
	)
	req.Header.Set("MCP-Protocol-Version", "2025-11-25")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return perform(server, req)
}

func (f *fixture) call(token, name, args string, wantError bool) string {
	f.t.Helper()

	response := f.rpc(f.server, token, "tools/call", `{"name":"`+name+`","arguments":`+args+`}`)
	require.Equal(f.t, http.StatusOK, response.Code, response.Body.String())

	var envelope struct {
		Result *struct {
			IsError bool            `json:"isError"`
			Data    json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(f.t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.NotNil(f.t, envelope.Result, response.Body.String())
	require.Equal(f.t, wantError, envelope.Result.IsError, response.Body.String())

	return string(envelope.Result.Data)
}

func (f *fixture) profile(
	server http.Handler,
	method, path string,
	owner bool,
) *httptest.ResponseRecorder {
	f.t.Helper()

	raw := make([]byte, 32)
	user := f.user

	if !owner {
		raw[0] = 1
		user = []byte("other")
	}

	hash := sha256.Sum256(raw)
	f.exec(
		"INSERT OR IGNORE INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, unixepoch() + 3600)",
		hash[:],
		user,
	)
	req := httptest.NewRequestWithContext(
		f.t.Context(),
		method,
		origin+"/api/connected-apps"+path,
		nil,
	)
	req.AddCookie(
		&http.Cookie{Name: "laga_session", Value: base64.RawURLEncoding.EncodeToString(raw)},
	)

	return perform(server, req)
}

func TestOAuthGrantLifecycle(t *testing.T) {
	f := newFixture(t)
	unsigned := f.rpc(f.server, "", "tools/list", "{}")
	require.Equal(t, http.StatusUnauthorized, unsigned.Code)
	assert.Contains(
		t,
		unsigned.Header().Get("WWW-Authenticate"),
		"/.well-known/oauth-protected-resource/mcp",
	)

	code := f.approve()
	form := codeForm(code)
	form.Set("code_verifier", strings.Repeat("Z", 43))
	assert.JSONEq(t, `{"error":"invalid_grant"}`, f.token(f.server, form).Body.String())
	form = codeForm(code)
	form.Set("resource", "https://other.example/mcp")
	assert.JSONEq(t, `{"error":"invalid_target"}`, f.token(f.server, form).Body.String())
	fresh := f.decodeTokens(f.token(f.server, codeForm(code)))
	require.NotEmpty(t, fresh.Refresh)

	rotated := f.decodeTokens(f.token(f.server, refreshForm(fresh.Refresh)))
	assert.NotEqual(t, fresh.Refresh, rotated.Refresh)
	assert.Equal(t, http.StatusOK, f.rpc(f.server, fresh.Access, "tools/list", "{}").Code)
	assert.JSONEq(
		t,
		`{"error":"invalid_grant"}`,
		f.token(f.server, refreshForm(fresh.Refresh)).Body.String(),
	)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.rpc(f.server, rotated.Access, "tools/list", "{}").Code,
	)
}

func TestAuthorizationErrorsStayOnHandlat(t *testing.T) {
	f := newFixture(t)
	query := authorizationQuery(redirectURI)
	query.Set("code_challenge_method", "plain")

	response := f.authorize(query)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"error":"invalid_request"}`, response.Body.String())
	assert.Empty(t, response.Header().Get("Location"))
	assert.Empty(t, response.Header().Get("Set-Cookie"))
}

func TestAuthorizationRequiresDeclaredCodeFlow(t *testing.T) {
	for _, claims := range []string{
		`"grant_types":["client_credentials"]`,
		`"response_types":["token"]`,
		`"grant_types":[]`,
		`"response_types":null`,
	} {
		t.Run(claims, func(t *testing.T) {
			f := newFixture(t)
			f.useMetadata(`{` + metadataIdentity + `,` + claims + `}`)

			response := f.authorize(authorizationQuery(redirectURI))
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.JSONEq(t, `{"error":"unauthorized_client"}`, response.Body.String())
			assert.Empty(t, response.Header().Get("Location"))
		})
	}
}

func TestRefreshTokenRequiresDeclaredGrant(t *testing.T) {
	f := newFixture(t)
	f.useMetadata(`{` + metadataIdentity + `}`)

	response := f.token(f.server, codeForm(f.approve()))
	issued := f.decodeTokens(response)
	assert.NotContains(t, response.Body.String(), "refresh_token")
	assert.Equal(t, http.StatusOK, f.rpc(f.server, issued.Access, "tools/list", "{}").Code)

	f.exec("UPDATE mcp_access_tokens SET expires_at = unixepoch() - 1")
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.rpc(f.server, issued.Access, "tools/list", "{}").Code,
	)
	assert.JSONEq(
		t,
		`{"enabled":true,"apps":[]}`,
		f.profile(f.server, http.MethodGet, "", true).Body.String(),
	)
}

func TestApprovalShowsCallbackDestination(t *testing.T) {
	f := newFixture(t)
	path, browser := f.beginApproval(redirectURI)
	response := f.approvalRequest(http.MethodGet, path, browser, "")
	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{
		"name":"Example agent","domain":"client.example","callbackHost":"callback.example",
		"localCallback":false,"renewable":true
	}`, response.Body.String())

	f.useMetadata(strings.ReplaceAll(defaultMetadata, redirectURI, "http://127.0.0.1/callback"))
	path, browser = f.beginApproval("http://127.0.0.1:51005/callback")
	response = f.approvalRequest(http.MethodGet, path, browser, "")
	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{
		"name":"Example agent","domain":"client.example","callbackHost":"127.0.0.1:51005",
		"localCallback":true,"renewable":true
	}`, response.Body.String())
}

func TestAuthorizationCodeReplayRevokesGrant(t *testing.T) {
	f := newFixture(t)
	code := f.approve()
	issued := f.decodeTokens(f.token(f.server, codeForm(code)))
	require.NotEmpty(t, issued.Refresh)

	for key, value := range map[string]string{
		"code_verifier": strings.Repeat("Z", 43),
		"client_id":     "https://other.example/client.json",
		"redirect_uri":  "https://other.example/callback",
	} {
		form := codeForm(code)
		form.Set(key, value)
		response := f.token(f.server, form)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.JSONEq(t, `{"error":"invalid_grant"}`, response.Body.String())
		assert.Equal(t, http.StatusOK, f.rpc(f.server, issued.Access, "tools/list", "{}").Code)
	}

	replayed := f.token(f.server, codeForm(code))
	assert.Equal(t, http.StatusBadRequest, replayed.Code)
	assert.JSONEq(t, `{"error":"invalid_grant"}`, replayed.Body.String())
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.rpc(f.server, issued.Access, "tools/list", "{}").Code,
	)
	assert.JSONEq(
		t,
		`{"error":"invalid_grant"}`,
		f.token(f.server, refreshForm(issued.Refresh)).Body.String(),
	)
}

func TestExpiredSpentCodeKeepsLiveGrant(t *testing.T) {
	f := newFixture(t)
	code := f.approve()
	issued := f.decodeTokens(f.token(f.server, codeForm(code)))
	f.exec("UPDATE mcp_codes SET expires_at = unixepoch() - 1")

	assert.JSONEq(t, `{"error":"invalid_grant"}`, f.token(f.server, codeForm(code)).Body.String())
	assert.Contains(t, f.profile(f.server, http.MethodGet, "", true).Body.String(), "Example agent")
	f.beginApproval(redirectURI)
	assert.Equal(t, http.StatusOK, f.rpc(f.server, issued.Access, "tools/list", "{}").Code)
	assert.NotEmpty(t, f.decodeTokens(f.token(f.server, refreshForm(issued.Refresh))).Refresh)
}

func TestMCPMembershipAndRecipePermissions(t *testing.T) {
	f := newFixture(t)
	token := f.decodeTokens(f.token(f.server, codeForm(f.approve()))).Access
	f.exec("INSERT INTO lists (id, name, created_at, updated_at) VALUES ('family', 'Family', 1, 1)")
	f.exec("INSERT INTO list_members (list_id, user_id) VALUES ('family', ?)", []byte("other"))

	listed := f.rpc(f.server, token, "tools/list", "{}")
	require.Equal(t, http.StatusOK, listed.Code)
	assert.Contains(t, listed.Body.String(), `"name":"delete_recipe"`)
	assertToolAnnotations(t, listed)
	f.call(token, "get_shopping_list", `{"listId":"family"}`, true)
	f.call(token, "add_shopping_list_item", `{"listId":"family","text":"Milk"}`, true)

	f.exec("INSERT INTO list_members (list_id, user_id) VALUES ('family', ?)", f.user)
	assert.Contains(
		t,
		f.call(token, "add_shopping_list_item", `{"listId":"family","text":"Milk"}`, false),
		"Milk",
	)
	f.call(token, "set_default_shopping_list", `{"listId":"family"}`, false)
	assert.Contains(t, f.call(token, "list_shopping_lists", "{}", false), `"isDefault":true`)

	const (
		recipeID = "d14668eb-bb48-4f69-979e-a0c3e109c23b"
		draft    = `{"id":"` + recipeID + `","name":"Cake","portions":3,"ingredients":[{"name":"Flour","amount":"1","unit":"dl"}]}`
	)
	f.call(token, "create_recipe", draft, true)
	f.call(token, "enable_recipes", "{}", false)
	f.call(token, "create_recipe", draft, false)

	transfer := `{"listId":"family","recipeId":"` + recipeID + `","portions":1,"requestId":"069f0ad3-493d-4e6b-b5c4-3fe0c2e27e69"}`
	assert.Contains(
		t,
		f.call(token, "transfer_recipe_ingredients", transfer, false),
		`"amount":"0.333333"`,
	)
	assert.Contains(
		t,
		f.call(token, "transfer_recipe_ingredients", transfer, false),
		`"amount":"0.333333"`,
	)

	f.exec("DELETE FROM list_members WHERE list_id = 'family' AND user_id = ?", f.user)
	f.call(token, "get_shopping_list", `{"listId":"family"}`, true)
	assert.Contains(t, f.call(token, "get_recipe", `{"recipeId":"`+recipeID+`"}`, false), "Cake")
}

func TestPauseReplacementAndRevocation(t *testing.T) {
	f := newFixture(t)
	first := f.decodeTokens(f.token(f.server, codeForm(f.approve())))
	second := f.decodeTokens(f.token(f.server, codeForm(f.approve())))
	assert.Equal(t, http.StatusUnauthorized, f.rpc(f.server, first.Access, "tools/list", "{}").Code)

	paused := f.handler(origin, false)
	assert.Equal(t, http.StatusNotFound, f.rpc(paused, second.Access, "tools/list", "{}").Code)
	assert.Equal(t, http.StatusNotFound, f.token(paused, refreshForm(second.Refresh)).Code)
	assert.Equal(t, http.StatusOK, f.rpc(f.server, second.Access, "tools/list", "{}").Code)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.rpc(f.handler("https://new.example", true), second.Access, "tools/list", "{}").Code,
	)

	profile := f.profile(paused, http.MethodGet, "", true)
	require.Equal(t, http.StatusOK, profile.Code)

	var apps struct {
		Enabled bool `json:"enabled"`
		Apps    []struct {
			ID string `json:"id"`
		} `json:"apps"`
	}
	require.NoError(t, json.Unmarshal(profile.Body.Bytes(), &apps))
	assert.False(t, apps.Enabled)
	require.Len(t, apps.Apps, 1)

	path := "/" + apps.Apps[0].ID
	assert.Equal(t, http.StatusNoContent, f.profile(paused, http.MethodDelete, path, false).Code)
	assert.Equal(t, http.StatusOK, f.rpc(f.server, second.Access, "tools/list", "{}").Code)
	assert.Equal(t, http.StatusNoContent, f.profile(paused, http.MethodDelete, path, true).Code)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.rpc(f.server, second.Access, "tools/list", "{}").Code,
	)
	assert.JSONEq(
		t,
		`{"error":"invalid_grant"}`,
		f.token(f.server, refreshForm(second.Refresh)).Body.String(),
	)
}

func TestInvalidClientMetadataReturnsLocalError(t *testing.T) {
	f := newFixture(t)
	f.useMetadata("")

	query := authorizationQuery(redirectURI)
	query.Set("client_id", "https://127.0.0.1/client.json")
	response := f.authorize(query)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, response.Header().Get("Location"))
}

func assertToolAnnotations(t *testing.T, result *httptest.ResponseRecorder) {
	t.Helper()

	var response struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Annotations struct {
					Destructive bool `json:"destructiveHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &response))

	annotations := map[string]bool{}
	for _, tool := range response.Result.Tools {
		annotations[tool.Name] = tool.Annotations.Destructive
	}

	assert.Contains(t, annotations, "clear_checked_items")
	assert.True(t, annotations["clear_checked_items"])
	assert.True(t, annotations["delete_recipe"])
	assert.False(t, annotations["add_shopping_list_item"])
}

func TestCurrentMCPProtocolVersion(t *testing.T) {
	f := newFixture(t)
	token := f.decodeTokens(f.token(f.server, codeForm(f.approve()))).Access
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
		"name":"list_shopping_lists","arguments":{},"_meta":{
			"io.modelcontextprotocol/protocolVersion":"2026-07-28",
			"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},
			"io.modelcontextprotocol/clientCapabilities":{}
		}}}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		resource,
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/call")
	req.Header.Set("Mcp-Name", "list_shopping_lists")
	response := perform(f.server, req)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"structuredContent":{"result":[]}`)
}

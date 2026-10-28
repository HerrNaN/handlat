package auth_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

func TestRegistrationCrossOriginProtection(t *testing.T) {
	conn, err := database.Open(t.Context(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))

	handler, err := auth.New(conn, "localhost", "http://localhost:8080", slog.Default())
	require.NoError(t, err)

	mux := http.NewServeMux()
	handler.Register(mux)
	api := http.NewCrossOriginProtection().Handler(http.StripPrefix("/api", mux))

	for _, tc := range []struct {
		name, origin, site string
		status             int
	}{
		{"cross-origin", "https://untrusted.example", "cross-site", http.StatusForbidden},
		{"same-origin", "http://localhost:8080", "same-origin", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"http://localhost:8080/api/auth/register/begin",
				nil,
			)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Sec-Fetch-Site", tc.site)

			res := httptest.NewRecorder()

			api.ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code)
		})
	}
}

package auth

import (
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/herrnan/laga/internal/app/auth/db"
	"github.com/herrnan/laga/internal/app/database"
	"github.com/herrnan/laga/internal/app/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteQueries(t *testing.T) {
	ctx := t.Context()

	conn, err := database.Open(ctx, filepath.Join(t.TempDir(), "laga.sqlite"))

	require.NoError(t, err)

	t.Cleanup(func() {
		err := conn.Close()

		assert.NoError(t, err)
	})

	err = migrations.Up(conn)

	require.NoError(t, err)

	queries := db.New(conn)
	userID := []byte{0, 255, 1}
	credentialID := []byte{255, 0, 2}
	const credential = `{"id":"credential","signCount":0}`
	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()

	t.Run("account transaction and JSON round trip", func(t *testing.T) {
		tx, err := conn.BeginTx(ctx, nil)

		require.NoError(t, err)

		t.Cleanup(func() { _ = tx.Rollback() })

		q := queries.WithTx(tx)

		err = q.InsertUser(ctx, db.InsertUserParams{ID: userID, Name: "Test user"})

		require.NoError(t, err)

		err = q.InsertCredential(ctx, db.InsertCredentialParams{
			ID:     credentialID,
			UserID: userID,
			Data:   credential,
		})

		require.NoError(t, err)

		err = q.InsertSession(ctx, db.InsertSessionParams{
			TokenHash: []byte("session"),
			UserID:    userID,
			ExpiresAt: future,
		})

		require.NoError(t, err)

		err = tx.Commit()

		require.NoError(t, err)

		account, err := queries.GetUserBySession(ctx, []byte("session"))

		require.NoError(t, err)
		assert.Equal(t, userID, account.ID)
		assert.Equal(t, "Test user", account.Name)

		stored, err := queries.GetUserByCredential(
			ctx,
			db.GetUserByCredentialParams{CredentialID: credentialID, UserID: userID},
		)

		require.NoError(t, err)
		assert.JSONEq(t, credential, stored.Data)

		err = queries.UpdateCredential(ctx, db.UpdateCredentialParams{
			ID:     credentialID,
			UserID: userID,
			Data:   `{"signCount":1}`,
		})

		require.NoError(t, err)

		stored, err = queries.GetUserByCredential(
			ctx,
			db.GetUserByCredentialParams{CredentialID: credentialID, UserID: userID},
		)

		require.NoError(t, err)
		assert.JSONEq(t, `{"signCount":1}`, stored.Data)
	})

	t.Run("constraints and rollback", func(t *testing.T) {
		err := queries.InsertUser(ctx, db.InsertUserParams{ID: userID, Name: "Duplicate"})

		require.Error(t, err)

		err = queries.InsertCredential(
			ctx,
			db.InsertCredentialParams{ID: []byte("other"), UserID: userID, Data: credential},
		)

		require.Error(t, err)

		err = queries.InsertSession(
			ctx,
			db.InsertSessionParams{
				TokenHash: []byte("orphan"),
				UserID:    []byte("missing"),
				ExpiresAt: future,
			},
		)

		require.Error(t, err)

		tx, err := conn.BeginTx(ctx, nil)

		require.NoError(t, err)

		handler := &Handler{log: slog.Default()}
		t.Cleanup(func() { handler.rollback(tx) })

		q := queries.WithTx(tx)

		err = q.InsertUser(ctx, db.InsertUserParams{ID: []byte("rolled-back"), Name: "Rollback"})

		require.NoError(t, err)

		err = q.InsertCredential(
			ctx,
			db.InsertCredentialParams{
				ID:     credentialID,
				UserID: []byte("rolled-back"),
				Data:   credential,
			},
		)

		require.Error(t, err)

		handler.rollback(tx)

		var count int
		err = conn.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id = ?", []byte("rolled-back")).
			Scan(&count)

		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("expired sessions are rejected", func(t *testing.T) {
		err := queries.InsertSession(ctx, db.InsertSessionParams{
			TokenHash: []byte("expired"),
			UserID:    userID,
			ExpiresAt: past,
		})

		require.NoError(t, err)

		_, err = queries.GetUserBySession(ctx, []byte("expired"))

		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("registration challenges expire and are consumed once", func(t *testing.T) {
		for _, expiry := range []int64{future, past} {
			hash := []byte(time.Unix(expiry, 0).String())

			err := queries.InsertRegistrationAttempt(ctx, db.InsertRegistrationAttemptParams{
				TokenHash: hash,
				UserID:    userID,
				Name:      "Test user",
				Ceremony:  `{"challenge":"test"}`,
				ExpiresAt: expiry,
			})

			require.NoError(t, err)

			attempt, err := queries.PopRegistrationAttempt(ctx, hash)

			if expiry == future {
				require.NoError(t, err)
				assert.Equal(t, userID, attempt.UserID)
				assert.JSONEq(t, `{"challenge":"test"}`, attempt.Ceremony)
			} else {
				require.ErrorIs(t, err, sql.ErrNoRows)
			}

			_, err = queries.PopRegistrationAttempt(ctx, hash)

			require.ErrorIs(t, err, sql.ErrNoRows)
		}
	})

	t.Run("login challenges expire and are consumed once under concurrency", func(t *testing.T) {
		err := queries.InsertLoginAttempt(ctx, db.InsertLoginAttemptParams{
			TokenHash: []byte("expired"),
			Ceremony:  `{}`,
			ExpiresAt: past,
		})

		require.NoError(t, err)

		_, err = queries.PopLoginAttempt(ctx, []byte("expired"))

		require.ErrorIs(t, err, sql.ErrNoRows)

		err = queries.InsertLoginAttempt(ctx, db.InsertLoginAttemptParams{
			TokenHash: []byte("concurrent"),
			Ceremony:  `{"challenge":"test"}`,
			ExpiresAt: future,
		})

		require.NoError(t, err)

		var wg sync.WaitGroup
		results := make(chan error, 8)

		for range 8 {
			wg.Go(func() {
				_, err := queries.PopLoginAttempt(ctx, []byte("concurrent"))
				results <- err
			})
		}

		wg.Wait()
		close(results)

		successes := 0

		for err := range results {
			if err == nil {
				successes++
			} else {
				assert.True(t, errors.Is(err, sql.ErrNoRows), "unexpected error: %v", err)
			}
		}

		assert.Equal(t, 1, successes)
	})
}

package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

func ListenAndServe(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		// Shutdown must outlive cancellation while retaining the parent context values.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	eg.Go(func() error {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}

		return nil
	})

	return eg.Wait()
}

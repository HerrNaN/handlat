package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/list"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/herrnan/handlat/internal/app/web"
	"github.com/herrnan/handlat/internal/util"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, log *slog.Logger, cfg Config) error {
	if cfg.DatabasePath == "" {
		return errors.New("missing database path")
	}

	db, err := database.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Close database", "error", err)
		}
	}()

	if err := migrations.Up(db); err != nil {
		return err
	}

	authHandler, err := auth.New(db, cfg.WebAuthnRPID, cfg.WebAuthnOrigin, log)
	if err != nil {
		return fmt.Errorf("configure authentication: %w", err)
	}

	mux := http.NewServeMux()

	apiMux := http.NewServeMux()
	authHandler.Register(apiMux)
	list.New(db, log).Register(apiMux, authHandler.RequireSession)

	mux.Handle("/api/", loggingMiddleware(http.NewCrossOriginProtection().Handler(http.StripPrefix("/api", apiMux))))

	if cfg.ExternalWebServerAddress != "" {
		serveWeb, err := web.ServeProxy(cfg.ExternalWebServerAddress)
		if err != nil {
			return fmt.Errorf("create web server proxy: %w", err)
		}

		log.Info(
			"Running against external web server",
			"external_web_server_address",
			cfg.ExternalWebServerAddress,
		)
		mux.HandleFunc("/", serveWeb)
	} else {
		serveWeb, err := web.ServeStatic()
		if err != nil {
			return fmt.Errorf("create static web server: %w", err)
		}

		mux.HandleFunc("/", serveWeb)
	}

	server := &http.Server{
		Handler:           http.MaxBytesHandler(mux, 1<<20 /* 1mb */),
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		ReadHeaderTimeout: 1 * time.Second,
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		log.Info("Server starting", "address", server.Addr)

		err := util.ListenAndServe(ctx, server, time.Second*10)
		if err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}

		return nil
	})

	return eg.Wait()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		if recorder.statusCode >= 300 {
			slog.Warn(
				"Non-2xx status code",
				"code",
				recorder.statusCode,
				"method",
				r.Method,
				"path",
				r.URL.Path,
			)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode      int
	wroteStatusCode bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if sr.wroteStatusCode {
		return
	}
	sr.wroteStatusCode = true
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if !sr.wroteStatusCode {
		sr.WriteHeader(http.StatusOK)
	}
	return sr.ResponseWriter.Write(b)
}

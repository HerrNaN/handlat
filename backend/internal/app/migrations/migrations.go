package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

func Up(db *sql.DB) error {
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

package migrations_test

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/pressly/goose/v3"
)

func requireRecipeMigrationOK(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func TestRecipeIngredientMigration(t *testing.T) {
	ctx := t.Context()
	conn, err := database.Open(ctx, filepath.Join(t.TempDir(), "recipe-ingredients.sqlite"))
	requireRecipeMigrationOK(t, err)
	t.Cleanup(func() { requireRecipeMigrationOK(t, conn.Close()) })

	requireRecipeMigrationOK(t, migrations.Up(conn))
	requireRecipeMigrationOK(t, goose.DownTo(conn, ".", 6))

	_, err = conn.ExecContext(ctx, "INSERT INTO users (id, name) VALUES (x'01', 'Owner')")
	requireRecipeMigrationOK(t, err)
	requireRecipeMigrationOK(t, migrations.Up(conn))

	var enabled bool

	err = conn.QueryRowContext(ctx, "SELECT recipes_enabled FROM users WHERE id = x'01'").Scan(&enabled)
	requireRecipeMigrationOK(t, err)

	if enabled {
		t.Fatal("recipes enabled by default for an existing user")
	}

	_, err = conn.ExecContext(ctx, `
		INSERT INTO recipe_collections (id, user_id) VALUES (1, x'01');
		INSERT INTO recipes (id, collection_id, name, portions) VALUES ('recipe', 1, 'Soup', 4);
		INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit) VALUES
		    ('recipe', 0, 'Stock', '9007199254740993.1234567890', 'ml'),
		    ('recipe', 100, 'Onions', '02', NULL),
		    ('recipe', 1, 'Basil', NULL, NULL),
		    ('recipe', 2, 'Pasta', NULL, 'pkt');
	`)
	requireRecipeMigrationOK(t, err)

	type ingredient struct {
		name         string
		amount, unit sql.NullString
	}

	want := []ingredient{
		{
			"Stock",
			sql.NullString{String: "9007199254740993.1234567890", Valid: true},
			sql.NullString{String: "ml", Valid: true},
		},
		{"Basil", sql.NullString{}, sql.NullString{}},
		{"Pasta", sql.NullString{}, sql.NullString{String: "pkt", Valid: true}},
		{"Onions", sql.NullString{String: "02", Valid: true}, sql.NullString{}},
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT name, amount, unit FROM recipe_ingredients WHERE recipe_id = 'recipe' ORDER BY position
	`)

	requireRecipeMigrationOK(t, err)
	defer func() { requireRecipeMigrationOK(t, rows.Close()) }()

	var got []ingredient

	for rows.Next() {
		var item ingredient

		requireRecipeMigrationOK(t, rows.Scan(&item.name, &item.amount, &item.unit))
		got = append(got, item)
	}

	requireRecipeMigrationOK(t, rows.Err())

	if !slices.Equal(got, want) {
		t.Fatalf("stored ingredients = %+v, want %+v", got, want)
	}

	_, err = conn.ExecContext(ctx, `
		INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)
		VALUES ('recipe', 4, 'Pepper', '1.5', 'cup')
	`)
	if err == nil || !strings.Contains(err.Error(), "recipe_ingredient_supported_unit") {
		t.Fatalf("expected unit constraint violation, got %v", err)
	}

	_, err = conn.ExecContext(ctx, "DELETE FROM recipes WHERE id = 'recipe'")
	requireRecipeMigrationOK(t, err)

	var count int

	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM recipe_ingredients").Scan(&count)
	requireRecipeMigrationOK(t, err)

	if count != 0 {
		t.Fatalf("found %d ingredients after deleting the recipe", count)
	}
}

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
	requireRecipeMigrationOK(t, goose.DownTo(conn, ".", 7))

	const original = `[{"name":"Stock","amount":"9007199254740993.1234567890","unit":"ml"},{"name":"Basil"},{"name":"Pasta","unit":"pkt"},{"name":"Onions","amount":"02"}]`

	_, err = conn.ExecContext(ctx, `
		INSERT INTO users (id, name) VALUES (x'01', 'Owner');
		INSERT INTO recipe_collections (id, user_id) VALUES (1, x'01');
	`)
	requireRecipeMigrationOK(t, err)

	_, err = conn.ExecContext(ctx, `
		INSERT INTO recipes (id, collection_id, name, portions, ingredients)
		VALUES ('recipe', 1, 'Soup', 4, ?)
	`, original)
	requireRecipeMigrationOK(t, err)
	requireRecipeMigrationOK(t, migrations.Up(conn))

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
		t.Fatalf("migrated ingredients = %+v, want %+v", got, want)
	}

	for _, amount := range []any{
		"", "0", "000.00", "-1", ".5", "5.", "1.2.3", "1,5", "1e3", " 1", "1\x002",
		strings.Repeat("1", 101), []byte("5"),
	} {
		_, err := conn.ExecContext(ctx, `
			UPDATE recipe_ingredients SET amount = ? WHERE recipe_id = 'recipe' AND position = 0
		`, amount)
		if err == nil || !strings.Contains(err.Error(), "recipe_ingredient_positive_decimal") {
			t.Errorf("amount %q: expected decimal constraint violation, got %v", amount, err)
		}
	}

	_, err = conn.ExecContext(ctx, `
		INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)
		VALUES ('recipe', 4, 'Pepper', '1.5', 'cup')
	`)
	if err == nil || !strings.Contains(err.Error(), "recipe_ingredient_supported_unit") {
		t.Fatalf("expected unit constraint violation, got %v", err)
	}

	requireRecipeMigrationOK(t, goose.DownTo(conn, ".", 7))

	var restored string

	err = conn.QueryRowContext(ctx, "SELECT ingredients FROM recipes WHERE id = 'recipe'").
		Scan(&restored)
	requireRecipeMigrationOK(t, err)

	if restored != original {
		t.Fatalf("restored ingredients = %s, want %s", restored, original)
	}

	requireRecipeMigrationOK(t, migrations.Up(conn))

	_, err = conn.ExecContext(ctx, "DELETE FROM recipes WHERE id = 'recipe'")
	requireRecipeMigrationOK(t, err)

	var count int

	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM recipe_ingredients").Scan(&count)
	requireRecipeMigrationOK(t, err)

	if count != 0 {
		t.Fatalf("found %d ingredients after deleting the recipe", count)
	}
}

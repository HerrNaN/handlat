-- +goose Up
CREATE TABLE recipe_ingredients (
    recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (typeof(position) = 'integer' AND position BETWEEN 0 AND 99),
    name TEXT NOT NULL CHECK (name <> ''),
    -- Decimal text preserves quantities exactly across reads and writes.
    amount TEXT CONSTRAINT recipe_ingredient_positive_decimal CHECK (
        amount IS NULL OR (
            typeof(amount) = 'text'
            AND instr(amount, char(0)) = 0
            AND length(amount) BETWEEN 1 AND 100
            AND amount GLOB '[0-9]*'
            AND amount GLOB '*[0-9]'
            AND amount NOT GLOB '*[^0-9.]*'
            AND amount NOT GLOB '*.*.*'
            AND amount GLOB '*[1-9]*'
        )
    ),
    unit TEXT CONSTRAINT recipe_ingredient_supported_unit CHECK (
        unit IS NULL OR unit IN ('st', 'pkt', 'kg', 'g', 'l', 'dl', 'cl', 'ml', 'msk', 'tsk', 'krm')
    ),
    PRIMARY KEY (recipe_id, position)
);

INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)
SELECT r.id, i.key, json_extract(i.value, '$.name'),
       json_extract(i.value, '$.amount'), json_extract(i.value, '$.unit')
FROM recipes AS r
CROSS JOIN json_each(r.ingredients) AS i;

ALTER TABLE recipes DROP COLUMN ingredients;

-- +goose Down
CREATE TABLE recipes_with_json (
    id TEXT PRIMARY KEY NOT NULL,
    collection_id INTEGER NOT NULL REFERENCES recipe_collections(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    portions INTEGER NOT NULL CHECK (portions BETWEEN 1 AND 9007199254740991),
    ingredients TEXT NOT NULL CHECK (json_valid(ingredients) AND json_type(ingredients) = 'array')
);

INSERT INTO recipes_with_json (id, collection_id, name, portions, ingredients)
SELECT r.id, r.collection_id, r.name, r.portions, (
    SELECT json_group_array(
        json_patch(json_object('name', i.name), json_object('amount', i.amount, 'unit', i.unit))
        ORDER BY i.position
    )
    FROM recipe_ingredients AS i
    WHERE i.recipe_id = r.id
)
FROM recipes AS r;

DROP TABLE recipe_ingredients;
DROP TABLE recipes;
ALTER TABLE recipes_with_json RENAME TO recipes;

CREATE INDEX recipes_collection ON recipes(collection_id, name, id);

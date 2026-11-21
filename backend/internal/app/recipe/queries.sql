-- name: RecipesEnabled :one
SELECT recipes_enabled FROM users WHERE id = @user_id;

-- name: EnableRecipes :exec
UPDATE users SET recipes_enabled = 1 WHERE id = @user_id;

-- name: EnsureCollection :exec
INSERT INTO recipe_collections (user_id) VALUES (@user_id)
ON CONFLICT (user_id) DO NOTHING;

-- name: ListRecipes :many
SELECT r.id, r.name FROM recipes AS r
JOIN recipe_collections AS c ON c.id = r.collection_id
WHERE c.user_id = @user_id
ORDER BY r.name COLLATE NOCASE, r.id;

-- name: GetRecipe :many
SELECT r.id, r.name, r.portions, i.name AS ingredient_name, i.amount, i.unit
FROM recipes AS r
JOIN recipe_collections AS c ON c.id = r.collection_id
LEFT JOIN recipe_ingredients AS i ON i.recipe_id = r.id
WHERE r.id = @id AND c.user_id = @user_id
ORDER BY i.position;

-- name: CreateRecipe :execrows
INSERT INTO recipes (id, collection_id, name, portions)
SELECT @id, c.id, @name, @portions
FROM recipe_collections AS c WHERE c.user_id = @user_id
ON CONFLICT (id) DO NOTHING;

-- name: UpdateRecipe :execrows
UPDATE recipes SET name = @name, portions = @portions
WHERE recipes.id = @id AND recipes.collection_id = (SELECT c.id FROM recipe_collections AS c WHERE c.user_id = @user_id);

-- name: AddRecipeIngredient :exec
INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)
VALUES (@recipe_id, @position, @name, @amount, @unit);

-- name: ClearRecipeIngredients :exec
DELETE FROM recipe_ingredients WHERE recipe_id = @recipe_id;

-- name: DeleteRecipe :execrows
DELETE FROM recipes
WHERE recipes.id = @id AND recipes.collection_id = (SELECT c.id FROM recipe_collections AS c WHERE c.user_id = @user_id);

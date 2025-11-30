-- name: CheckUsersTableExists :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      information_schema.tables
    WHERE
      table_schema = 'public'
      AND table_name = 'users');

-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password_hash, role)
  VALUES (trim(lower(@email::text)), $1, $2, $3, 'user')
RETURNING
  id;

-- name: CreateAdmin :one
INSERT INTO users (email, first_name, last_name, password_hash, role)
  VALUES (trim(lower(@email::text)), $1, $2, $3, 'admin')
RETURNING
  id;

-- name: GetAdminCount :one
SELECT
  count(*)
FROM
  users
WHERE
  ROLE = 'admin';

-- name: GetUser :one
SELECT
  id,
  password_hash,
  ROLE
FROM
  users
WHERE
  email = trim(lower($1));

-- name: GetUserRefreshTokenHash :one
SELECT
  refresh_token_hash,
  refresh_token_expires_at
FROM
  users
WHERE
  id = $1;

-- name: GetUserRole :one
SELECT
  ROLE
FROM
  users
WHERE
  id = $1;

-- name: UpdateUserRefreshTokenHash :exec
UPDATE
  users
SET
  refresh_token_hash = $1
WHERE
  id = $2;

-- name: CreateRecipe :one
INSERT INTO recipes (user_id, title)
  VALUES ($1, $2)
RETURNING
  id;

-- name: GetRecipeOwner :one
SELECT
  user_id
FROM
  recipes
WHERE
  id = $1;

-- name: CreateRecipeIngredient :one
INSERT INTO recipe_ingredients (recipe_id, quantity, unit, name, image_url)
  VALUES ($1, $2, $3, $4, $5)
RETURNING
  id;

-- name: UpdateRecipeIngredientImage :exec
UPDATE
  recipe_ingredients
SET
  image_url = $1
WHERE
  id = $2;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (recipe_id, instruction)
  VALUES ($1, $2)
RETURNING
  id;

-- name: UpdateRecipeStepImage :exec
UPDATE
  recipe_steps
SET
  image_url = $1
WHERE
  id = $2;

-- name: CheckRecipeOwnership :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipes
    WHERE
      id = $1
      AND user_id = $2);

-- name: UpdateRecipeCoverImage :exec
UPDATE
  recipes
SET
  image_url = $1
WHERE
  id = $2;

-- name: GetRecipeAndOwner :one
SELECT
  r.user_id,
  r.image_url,
  r.title,
  r.description,
  r.created_at,
  r.updated_at,
  r.published,
  r.cook_time_minutes,
  u.first_name,
  u.last_name,
  u.id
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  r.id = $1
  AND r.published = TRUE;

-- name: GetRecipeSteps :many
SELECT
  *
FROM
  recipe_steps
WHERE
  recipe_id = $1
ORDER BY
  step_number ASC;

-- name: GetRecipeIngredients :many
SELECT
  *
FROM
  recipe_ingredients
WHERE
  recipe_id = $1
ORDER BY
  created_at ASC;

-- name: GetRecipesByOwner :many
SELECT
  r.user_id,
  r.image_url,
  r.title,
  r.description,
  r.created_at,
  r.updated_at,
  r.published,
  r.cook_time_minutes,
  u.first_name,
  u.last_name,
  u.id
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  u.id = $1
ORDER BY
  r.updated_at DESC;

-- name: DeleteRecipe :exec
DELETE FROM recipes
WHERE id = $1;

-- name: DeleteRecipeIngredient :exec
DELETE FROM recipe_ingredients
WHERE id = $1;

-- name: GetRecipeIngredientImageURL :one
SELECT
  image_url
FROM
  recipe_ingredients
WHERE
  id = $1;

-- name: GetRecipeIngredientExistance :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipe_ingredients
    WHERE
      id = $1);

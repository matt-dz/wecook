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
  r.id,
  r.cook_time_minutes,
  u.first_name,
  u.last_name,
  u.id
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  r.id = $1;

-- name: GetPublishedRecipeAndOwner :one
SELECT
  r.user_id,
  r.image_url,
  r.title,
  r.description,
  r.created_at,
  r.updated_at,
  r.published,
  r.id,
  r.cook_time_minutes,
  u.first_name,
  u.last_name,
  u.id
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  r.id = $1
  AND published = TRUE;

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
  r.id,
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

-- name: GetRecipeIngredientExistence :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipe_ingredients
    WHERE
      id = $1);

-- name: GetRecipeStepExistence :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipe_steps
    WHERE
      id = $1);

-- name: GetRecipeStepImageURL :one
SELECT
  image_url
FROM
  recipe_steps
WHERE
  id = $1;

-- name: DeleteRecipeStep :exec
DELETE FROM recipe_steps
WHERE id = $1;

-- name: UpdateRecipeStep :exec
UPDATE
  recipe_steps
SET
  instruction = coalesce(sqlc.narg ('instruction'), instruction),
  image_url = coalesce(sqlc.narg ('image_url'), image_url)
WHERE
  id = $1;

-- name: UpdateRecipeIngredient :exec
UPDATE
  recipe_ingredients
SET
  quantity = coalesce(sqlc.narg ('quantity'), quantity),
  unit = coalesce(sqlc.narg ('unit'), unit),
  name = coalesce(sqlc.narg ('name'), name),
  image_url = coalesce(sqlc.narg ('image_url'), image_url)
WHERE
  id = $1;

-- name: UpdateRecipe :exec
UPDATE
  recipes
SET
  image_url = coalesce(sqlc.narg ('image_url'), image_url),
  title = coalesce(sqlc.narg ('title'), title),
  description = coalesce(sqlc.narg ('description'), description),
  published = coalesce(sqlc.narg ('published'), published),
  cook_time_minutes = coalesce(sqlc.narg ('cook_time_minutes'), cook_time_minutes)
WHERE
  id = $1;

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

-- name: CreateEmptyRecipeIngredient :one
INSERT INTO recipe_ingredients (recipe_id)
  VALUES ($1)
RETURNING
  *;

-- name: UpdateRecipeIngredientImage :exec
UPDATE
  recipe_ingredients
SET
  image_url = $1
WHERE
  id = $2;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (recipe_id, instruction)
  VALUES ($1, sqlc.narg ('instruction'))
RETURNING
  id, step_number;

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

-- name: CheckIngredientOwnership :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipe_ingredients ri
      JOIN recipes r ON r.id = ri.recipe_id
    WHERE
      r.id = @recipe_id::bigint
      AND ri.id = @ingredient_id::bigint
      AND r.user_id = $1);

-- name: CheckStepOwnership :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      recipe_steps rs
      JOIN recipes r ON r.id = rs.recipe_id
    WHERE
      r.id = @recipe_id::bigint
      AND rs.id = @step_id::bigint
      AND r.user_id = $1);

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
  r.cook_time_amount,
  r.cook_time_unit,
  r.prep_time_amount,
  r.prep_time_unit,
  r.servings,
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
  r.servings,
  r.cook_time_amount,
  r.cook_time_unit,
  r.prep_time_amount,
  r.prep_time_unit,
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
  r.cook_time_amount,
  r.cook_time_unit,
  r.prep_time_amount,
  r.prep_time_unit,
  r.id AS recipe_id,
  r.servings,
  u.first_name,
  u.last_name
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  u.id = $1
ORDER BY
  r.updated_at DESC;

-- name: GetPublicRecipes :many
SELECT
  r.user_id,
  r.image_url,
  r.title,
  r.description,
  r.created_at,
  r.updated_at,
  r.published,
  r.cook_time_amount,
  r.cook_time_unit,
  r.prep_time_amount,
  r.prep_time_unit,
  r.id AS recipe_id,
  r.servings,
  u.first_name,
  u.last_name
FROM
  recipes r
  JOIN users u ON r.user_id = u.id
WHERE
  r.published = TRUE
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

-- name: UpdateRecipeStep :one
UPDATE
  recipe_steps
SET
  instruction = CASE WHEN sqlc.narg ('update_instruction')::boolean THEN
    sqlc.narg ('instruction')
  ELSE
    instruction
  END,
  step_number = CASE WHEN sqlc.narg ('update_step_number')::boolean THEN
    sqlc.narg ('step_number')
  ELSE
    step_number
  END,
  image_url = CASE WHEN sqlc.narg ('update_image_url')::boolean THEN
    sqlc.narg ('image_url')
  ELSE
    image_url
  END
WHERE
  id = $1
RETURNING
  id,
  instruction,
  step_number,
  image_url;

-- name: UpdateRecipeIngredient :one
UPDATE
  recipe_ingredients
SET
  quantity = CASE WHEN sqlc.narg ('update_quantity')::boolean THEN
    sqlc.narg ('quantity')
  ELSE
    quantity
  END,
  unit = CASE WHEN sqlc.narg ('update_unit')::boolean THEN
    sqlc.narg ('unit')
  ELSE
    unit
  END,
  name = CASE WHEN sqlc.narg ('update_name')::boolean THEN
    sqlc.narg ('name')
  ELSE
    name
  END,
  image_url = CASE WHEN sqlc.narg ('update_image_url')::boolean THEN
    sqlc.narg ('image_url')
  ELSE
    image_url
  END
WHERE
  id = $1
RETURNING
  *;

-- name: UpdateRecipe :one
UPDATE
  recipes
SET
  image_url = CASE WHEN sqlc.narg ('update_image_url')::boolean THEN
    sqlc.narg ('image_url')
  ELSE
    image_url
  END,
  title = CASE WHEN sqlc.narg ('update_title')::boolean THEN
    sqlc.narg ('title')
  ELSE
    title
  END,
  description = CASE WHEN sqlc.narg ('update_description')::boolean THEN
    sqlc.narg ('description')
  ELSE
    description
  END,
  published = CASE WHEN sqlc.narg ('update_published')::boolean THEN
    sqlc.narg ('published')
  ELSE
    published
  END,
  cook_time_amount = CASE WHEN sqlc.narg ('update_cook_time_amount')::boolean THEN
    sqlc.narg ('cook_time_amount')
  ELSE
    cook_time_amount
  END,
  cook_time_unit = CASE WHEN sqlc.narg ('update_cook_time_unit')::boolean THEN
    sqlc.narg ('cook_time_unit')
  ELSE
    cook_time_unit
  END,
  prep_time_amount = CASE WHEN sqlc.narg ('update_prep_time_amount')::boolean THEN
    sqlc.narg ('prep_time_amount')
  ELSE
    prep_time_amount
  END,
  prep_time_unit = CASE WHEN sqlc.narg ('update_prep_time_unit')::boolean THEN
    sqlc.narg ('prep_time_unit')
  ELSE
    prep_time_unit
  END,
  servings = CASE WHEN sqlc.narg ('update_servings')::boolean THEN
    sqlc.narg ('servings')
  ELSE
    servings
  END
WHERE
  id = $1
RETURNING
  id,
  image_url,
  title,
  description,
  published,
  cook_time_amount,
  cook_time_unit,
  prep_time_amount,
  prep_time_unit,
  servings,
  updated_at,
  created_at;

-- name: GetRecipeIngredientIDs :many
SELECT
  id
FROM
  recipe_ingredients
WHERE
  recipe_id = $1;

-- name: GetRecipeStepIDs :many
SELECT
  id
FROM
  recipe_steps
WHERE
  recipe_id = $1;

-- name: DeleteRecipeIngredientsByIDs :exec
DELETE FROM recipe_ingredients
WHERE recipe_id = $1
  AND id = ANY (@ids::bigint[]);

-- name: DeleteRecipeStepsByIDs :exec
DELETE FROM recipe_steps
WHERE recipe_id = $1
  AND id = ANY (@ids::bigint[]);

-- name: BulkInsertRecipeIngredients :copyfrom
INSERT INTO recipe_ingredients (recipe_id, quantity, unit, name, image_url)
  VALUES ($1, $2, $3, $4, $5);

-- name: BulkInsertRecipeSteps :copyfrom
INSERT INTO recipe_steps (recipe_id, instruction, image_url, step_number)
  VALUES ($1, $2, $3, $4);

-- name: BulkUpdateRecipeIngredients :batchexec
UPDATE
  recipe_ingredients
SET
  quantity = $2,
  unit = $3,
  name = $4,
  image_url = $5
WHERE
  id = $1;

-- name: BulkUpdateRecipeSteps :batchexec
UPDATE
  recipe_steps
SET
  instruction = $2,
  image_url = $3
WHERE
  id = $1;

-- name: BatchUpdateRecipeIngredientImages :batchexec
UPDATE
  recipe_ingredients
SET
  image_url = $2
WHERE
  id = $1;

-- name: BatchUpdateRecipeStepImages :batchexec
UPDATE
  recipe_steps
SET
  image_url = $2
WHERE
  id = $1;

-- name: DeleteRecipeStepImageURL :exec
UPDATE
  recipe_steps
SET
  image_url = NULL
WHERE
  id = $1;

-- name: DeleteRecipeIngredientImageURL :exec
UPDATE
  recipe_ingredients
SET
  image_url = NULL
WHERE
  id = $1;

-- name: GetRecipeImageURL :one
SELECT
  image_url
FROM
  recipes
WHERE
  id = $1;

-- name: GetUsers :many
SELECT
  id,
  email,
  first_name,
  last_name,
  ROLE
FROM
  users
WHERE
  id > coalesce(sqlc.narg ('after'), 0)
ORDER BY
  id
LIMIT LEAST (100, GREATEST (1, coalesce(sqlc.narg ('limit')::int, 20)));

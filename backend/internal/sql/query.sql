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

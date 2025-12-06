CREATE TYPE ROLE AS enum (
  'admin',
  'user'
);

CREATE TYPE time_unit AS enum (
  'minutes',
  'hours',
  'days'
);

CREATE TABLE users (
  id bigserial PRIMARY KEY,
  email text NOT NULL,
  first_name text NOT NULL,
  last_name text NOT NULL,
  ROLE ROLE NOT NULL DEFAULT 'user',
  password_hash text NOT NULL,
  refresh_token_hash text,
  refresh_token_expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((refresh_token_hash IS NULL AND refresh_token_expires_at IS NULL) OR (refresh_token_hash IS NOT NULL AND
    refresh_token_expires_at IS NOT NULL))
);

CREATE UNIQUE INDEX users_unique_email ON users (trim(lower(email)))
WHERE
  email IS NOT NULL;

CREATE TABLE recipes (
  id bigserial PRIMARY KEY,
  user_id bigserial REFERENCES users (id) ON DELETE CASCADE,
  image_url text,
  title text NOT NULL,
  description text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  published bool NOT NULL DEFAULT FALSE,
  cook_time_amount int CHECK (cook_time_amount >= 0),
  cook_time_unit time_unit,
  prep_time_amount int CHECK (prep_time_amount >= 0),
  prep_time_unit time_unit,
  servings real CHECK (servings > 0)
);

CREATE TABLE recipe_ingredients (
  id bigserial PRIMARY KEY,
  recipe_id bigint NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
  quantity real NOT NULL CHECK (quantity > 0),
  unit text,
  name text NOT NULL,
  image_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE recipe_steps (
  id bigserial PRIMARY KEY,
  recipe_id bigint NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
  step_number int NOT NULL,
  instruction text NOT NULL,
  image_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (recipe_id, step_number)
);

CREATE OR REPLACE FUNCTION recipe_steps_before_insert ()
  RETURNS TRIGGER
  AS $$
DECLARE
  max_step int;
BEGIN
  -- If no step_number provided or invalid, append at the end
  IF NEW.step_number IS NULL OR NEW.step_number <= 0 THEN
    SELECT
      COALESCE(MAX(step_number), 0) INTO max_step
    FROM
      recipe_steps
    WHERE
      recipe_id = NEW.recipe_id;
    NEW.step_number := max_step + 1;
    RETURN NEW;
  END IF;
  -- Clamp step_number so it can't skip beyond the end
  SELECT
    COALESCE(MAX(step_number), 0) INTO max_step
  FROM
    recipe_steps
  WHERE
    recipe_id = NEW.recipe_id;
  IF NEW.step_number > max_step + 1 THEN
    NEW.step_number := max_step + 1;
  END IF;
  -- Shift existing steps at or after this position
  UPDATE
    recipe_steps
  SET
    step_number = step_number + 1
  WHERE
    recipe_id = NEW.recipe_id
    AND step_number >= NEW.step_number;
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER recipe_steps_before_insert_trg
  BEFORE INSERT ON recipe_steps
  FOR EACH ROW
  EXECUTE FUNCTION recipe_steps_before_insert ();

CREATE OR REPLACE FUNCTION recipe_steps_after_delete ()
  RETURNS TRIGGER
  AS $$
BEGIN
  UPDATE
    recipe_steps
  SET
    step_number = step_number - 1
  WHERE
    recipe_id = OLD.recipe_id
    AND step_number > OLD.step_number;
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER recipe_steps_after_delete_trg
  AFTER DELETE ON recipe_steps
  FOR EACH ROW
  EXECUTE FUNCTION recipe_steps_after_delete ();

CREATE FUNCTION update_table_updated_at ()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

CREATE TRIGGER users_set_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE PROCEDURE update_table_updated_at ();

CREATE TRIGGER recipes_set_updated_at
  BEFORE UPDATE ON recipes
  FOR EACH ROW
  EXECUTE PROCEDURE update_table_updated_at ();

CREATE TRIGGER recipe_steps_set_updated_at
  BEFORE UPDATE ON recipe_steps
  FOR EACH ROW
  EXECUTE PROCEDURE update_table_updated_at ();

CREATE TRIGGER recipe_ingredients_set_updated_at
  BEFORE UPDATE ON recipe_ingredients
  FOR EACH ROW
  EXECUTE PROCEDURE update_table_updated_at ();

CREATE OR REPLACE FUNCTION set_refresh_token_expiry ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF NEW.refresh_token_hash IS NOT NULL AND (OLD.refresh_token_hash IS DISTINCT FROM NEW.refresh_token_hash) THEN
    NEW.refresh_token_expires_at := now() + interval '14 days';
  END IF;
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER trg_set_refresh_token_expiry
  BEFORE UPDATE OF refresh_token_hash ON users
  FOR EACH ROW
  EXECUTE FUNCTION set_refresh_token_expiry ();

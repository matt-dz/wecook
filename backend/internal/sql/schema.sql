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

CREATE TABLE invitation_codes (
  id bigserial PRIMARY KEY,
  code_hash text NOT NULL,
  invited_by bigint REFERENCES users (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  expires_at timestamptz NOT NULL DEFAULT (now() + interval '8 hours'),
  used_at timestamptz
);

ALTER TABLE invitation_codes
  ADD CONSTRAINT invitation_codes_expires_after_created CHECK (expires_at >= created_at);

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
  quantity real CHECK (quantity > 0),
  unit text,
  name text,
  image_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE recipe_steps (
  id bigserial PRIMARY KEY,
  recipe_id bigint NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
  step_number int NOT NULL,
  instruction text,
  image_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (recipe_id, step_number) DEFERRABLE INITIALLY DEFERRED
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
  -- Set flag to indicate we're in a shift operation to prevent before_update trigger from running
  PERFORM
    set_config('recipe_steps.in_shift', '1', TRUE);
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

CREATE OR REPLACE FUNCTION recipe_steps_before_update ()
  RETURNS TRIGGER
  AS $$
DECLARE
  max_step int;
  in_shift text;
BEGIN
  -- Check if we're already in a shift operation to prevent recursion
  BEGIN
    in_shift := current_setting('recipe_steps.in_shift', TRUE);
  EXCEPTION
    WHEN OTHERS THEN
      in_shift := NULL;
  END;
  -- If we're in a shift operation, skip the shifting logic
  IF in_shift = '1' THEN
    RETURN NEW;
    END IF;
    -- Only handle step_number changes
    IF OLD.step_number = NEW.step_number THEN
      RETURN NEW;
      END IF;
      -- Clamp step_number to valid range
      SELECT
        COALESCE(MAX(step_number), 0) INTO max_step
      FROM
        recipe_steps
      WHERE
        recipe_id = NEW.recipe_id;
        IF NEW.step_number < 1 THEN
          NEW.step_number := 1;
        ELSIF NEW.step_number > max_step THEN
          NEW.step_number := max_step;
          END IF;
          -- Set flag to indicate we're now in a shift operation
          PERFORM
            set_config('recipe_steps.in_shift', '1', TRUE);
            -- If moving to a later position, decrement steps in between
            IF NEW.step_number > OLD.step_number THEN
              UPDATE
                recipe_steps
              SET
                step_number = step_number - 1
              WHERE
                recipe_id = NEW.recipe_id
                AND step_number > OLD.step_number
                AND step_number <= NEW.step_number
                AND id != NEW.id;
                -- If moving to an earlier position, increment steps in between
              ELSIF NEW.step_number < OLD.step_number THEN
                UPDATE
                  recipe_steps
                SET
                  step_number = step_number + 1
                WHERE
                  recipe_id = NEW.recipe_id
                  AND step_number >= NEW.step_number
                  AND step_number < OLD.step_number
                  AND id != NEW.id;
              END IF;
              RETURN NEW;
END;

$$
LANGUAGE plpgsql;

CREATE TRIGGER recipe_steps_before_update_trg
  BEFORE UPDATE ON recipe_steps
  FOR EACH ROW
  EXECUTE FUNCTION recipe_steps_before_update ();

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

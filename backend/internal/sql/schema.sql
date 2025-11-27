CREATE TYPE ROLE AS enum (
  'superuser',
  'admin',
  'user'
);

CREATE TABLE users (
	id bigserial PRIMARY KEY,
	email TEXT NOT NULL,
	first_name text NOT NULL,
	last_name text,
	role ROLE NOT NULL DEFAULT 'user'
);

CREATE TABLE recipes (
	id bigserial PRIMARY KEY,
	user_id bigserial REFERENCES users(id) ON DELETE CASCADE,
	image_url text,
	title text NOT NULL,
	description text,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
	updated_at TIMESTAMPTZ DEFAULT NOW (),
	published bool NOT NULL DEFAULT false,
	cook_time_minutes int CHECK (cook_time_minutes >= 0)
);

CREATE TABLE recipe_ingredients (
    recipe_id   bigint NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
		quantity int NOT NULL,
		unit text,
		name text NOT NULL
);

CREATE TABLE recipe_steps (
    recipe_id   bigint NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_number int NOT NULL,
    instruction text NOT NULL,
		image_url text,

    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    UNIQUE (recipe_id, step_number)
);

CREATE OR REPLACE FUNCTION recipe_steps_before_insert()
RETURNS trigger AS
$$
DECLARE
    max_step int;
BEGIN
    -- If no step_number provided or invalid, append at the end
    IF NEW.step_number IS NULL OR NEW.step_number <= 0 THEN
        SELECT COALESCE(MAX(step_number), 0)
        INTO max_step
        FROM recipe_steps
        WHERE recipe_id = NEW.recipe_id;

        NEW.step_number := max_step + 1;
        RETURN NEW;
    END IF;

    -- Clamp step_number so it can't skip beyond the end
    SELECT COALESCE(MAX(step_number), 0)
    INTO max_step
    FROM recipe_steps
    WHERE recipe_id = NEW.recipe_id;

    IF NEW.step_number > max_step + 1 THEN
        NEW.step_number := max_step + 1;
    END IF;

    -- Shift existing steps at or after this position
    UPDATE recipe_steps
    SET step_number = step_number + 1
    WHERE recipe_id = NEW.recipe_id
      AND step_number >= NEW.step_number;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER recipe_steps_before_insert_trg
BEFORE INSERT ON recipe_steps
FOR EACH ROW
EXECUTE FUNCTION recipe_steps_before_insert();

CREATE OR REPLACE FUNCTION recipe_steps_after_delete()
RETURNS trigger AS
$$
BEGIN
    UPDATE recipe_steps
    SET step_number = step_number - 1
    WHERE recipe_id = OLD.recipe_id
      AND step_number > OLD.step_number;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER recipe_steps_after_delete_trg
AFTER DELETE ON recipe_steps
FOR EACH ROW
EXECUTE FUNCTION recipe_steps_after_delete();

-- migrate:up
ALTER TABLE users ADD COLUMN birth_of_date BIGINT NOT NULL;

-- migrate:down
ALTER TABLE users DROP COLUMN birth_of_date;

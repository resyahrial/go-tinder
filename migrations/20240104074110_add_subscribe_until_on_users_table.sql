-- migrate:up
ALTER TABLE users ADD COLUMN subscribe_until BIGINT;

-- migrate:down
ALTER TABLE users DROP COLUMN subscribe_until;

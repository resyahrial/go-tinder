-- migrate:up
CREATE TABLE IF NOT EXISTS users (
  id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  created_at BIGINT NOT NULL DEFAULT DATE_PART('EPOCH', NOW()) ,
  updated_at BIGINT NOT NULL DEFAULT DATE_PART('EPOCH', NOW()),
  email VARCHAR(255),
  password VARCHAR(255) NOT NULL
);

-- migrate:down
DROP TABLE IF EXISTS users;

-- migrate:up
CREATE TABLE IF NOT EXISTS latest_locations (
  id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  updated_at BIGINT NOT NULL,
  lat REAL NOT NULL,
  lng REAL NOT NULL,
  location geography(Point, 4326) NOT NULL,
  user_id uuid NOT NULL,
  CONSTRAINT fk_users_latest_locations FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX uidx_latest_locations_user_id ON latest_locations(user_id);

-- migrate:down
DROP INDEX uidx_latest_locations_user_id;

DROP TABLE IF EXISTS latest_locations;

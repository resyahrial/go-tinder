-- migrate:up
CREATE TABLE IF NOT EXISTS coupons (
  id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  created_at BIGINT NOT NULL DEFAULT DATE_PART('EPOCH', NOW()),
  code VARCHAR(255) NOT NULL,
  duration_in_second INT NOT NULL,
  valid_until BIGINT NOT NULL
);

CREATE UNIQUE INDEX uidx_coupons_code ON coupons(code);

-- migrate:down
DROP INDEX uidx_coupons_code;

DROP TABLE IF EXISTS coupons;

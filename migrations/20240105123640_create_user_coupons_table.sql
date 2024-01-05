-- migrate:up
CREATE TABLE IF NOT EXISTS user_coupons (
  id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  created_at BIGINT NOT NULL DEFAULT DATE_PART('EPOCH', NOW()),
  user_id uuid NOT NULL,
  coupon_id uuid NOT NULL,
  used_at BIGINT,
  CONSTRAINT fk_users_user_coupons FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_coupon_user_coupons FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX uidx_user_coupons_user_id_coupons_id ON user_coupons(user_id, coupon_id) WHERE used_at IS NULL;

-- migrate:down
DROP INDEX uidx_user_coupons_user_id_coupon_id;

DROP TABLE IF EXISTS user_coupons;

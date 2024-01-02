-- migrate:up
CREATE TABLE IF NOT EXISTS likes (
    self_id uuid NOT NULL,
    target_id uuid NOT NULL,
    created_at BIGINT NOT NULL DEFAULT DATE_PART('EPOCH', NOW()),
    PRIMARY KEY(self_id, target_id),
    CONSTRAINT fk_users_likes_self FOREIGN KEY (self_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_users_likes_target FOREIGN KEY (target_id) REFERENCES users(id) ON DELETE CASCADE
);

-- migrate:down
DROP TABLE IF EXISTS likes;

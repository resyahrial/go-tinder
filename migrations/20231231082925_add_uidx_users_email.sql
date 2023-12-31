-- migrate:up
CREATE UNIQUE INDEX uidx_users_email ON users(email); 

-- migrate:down
DROP INDEX uidx_users_email;


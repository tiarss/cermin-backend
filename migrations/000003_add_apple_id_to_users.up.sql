ALTER TABLE users
ADD COLUMN apple_id VARCHAR(255) UNIQUE;

CREATE INDEX idx_users_apple_id ON users(apple_id);

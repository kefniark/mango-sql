CREATE TABLE users (
  id          UUID PRIMARY KEY,
  email       VARCHAR(64) NOT NULL,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP
);

CREATE TABLE items (
  id          UUID PRIMARY KEY,
  name        TEXT NOT NULL,
  quantity    INT NOT NULL,
  user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMP NOT NULL,
  updated_at  TIMESTAMP NOT NULL,
  deleted_at  TIMESTAMP
);

CREATE TABLE item_meta (
  id          UUID PRIMARY KEY,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  item_id     UUID REFERENCES items(id) ON DELETE CASCADE
);

INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test1@localhost', 'user1');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test2@localhost', 'user2');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test3@localhost', 'user3');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test4@localhost', 'user4');

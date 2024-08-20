CREATE TABLE users (
  id          INTEGER PRIMARY KEY,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP
);
CREATE TABLE users (
  id          UUID PRIMARY KEY,
  email       VARCHAR(64) NOT NULL,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP
);
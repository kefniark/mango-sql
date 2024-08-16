CREATE TABLE users (
  id          string PRIMARY KEY,
  email       VARCHAR(64) NOT NULL,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at  TIMESTAMP
);
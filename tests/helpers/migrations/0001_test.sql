-- +goose Up

CREATE TABLE users (
  id          UUID PRIMARY KEY,
  email       VARCHAR(64) NOT NULL,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP DEFAULT NULL
);

CREATE TABLE orders (
  id          UUID  PRIMARY KEY,
  name        text  NOT NULL,
  user_id     UUID  NOT NULL REFERENCES users(id),
  status      VARCHAR(32) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at  TIMESTAMP
);

CREATE TABLE products (
  id    UUID  PRIMARY KEY,
  name  text  NOT NULL
);

CREATE TABLE product_variations (
  id          UUID  PRIMARY KEY,
  name        text  NOT NULL,
  price       int,
  product_id  UUID REFERENCES products(id)
);

CREATE TABLE order_items (
  id        UUID  PRIMARY KEY,
  name      text  NOT NULL,
  quantity  int,
  price     int,
  order_id  UUID REFERENCES orders(id),
  product_variation_id UUID REFERENCES product_variations(id)
);

CREATE INDEX order_user_id ON orders(user_id);
CREATE INDEX order_status ON orders(status);
CREATE INDEX order_items_order_id ON order_items(order_id);
CREATE INDEX order_items_product_variation_id ON order_items(product_variation_id);

-- +goose Down
DROP TABLE users;
DROP TABLE orders;
DROP TABLE products;
DROP TABLE product_variations;
DROP TABLE order_items;

DROP INDEX order_user_id;
DROP INDEX order_status;
DROP INDEX order_items_order_id;
DROP INDEX order_items_product_variation_id;

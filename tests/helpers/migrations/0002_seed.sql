-- +goose Up

INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test1@localhost', 'user1');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test2@localhost', 'user2');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test3@localhost', 'user3');
INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), 'test4@localhost', 'user4');

-- +goose Down

DELETE FROM users;
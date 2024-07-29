-- queryMany: UserItems
SELECT *
FROM users as u
INNER JOIN items ON items.user_id = users.id
WHERE users.deleted_at IS NULL;

-- queryMany: UserItems2
SELECT users.*, items.id
FROM users
INNER JOIN items ON items.user_id = users.id
WHERE users.deleted_at IS NULL;

-- queryMany: UserItems3
SELECT *
FROM users
INNER JOIN items AS items1 ON i1.user_id = users.id
INNER JOIN items AS items2 ON i2.id = i1.id
INNER JOIN items AS items3 ON i3.id = i2.id
WHERE users.deleted_at IS NULL;

-- queryMany: UserItemsCount
SELECT users.*, count(items.id) as count, sum(items.quantity) as sum
FROM users
INNER JOIN items ON items.user_id = users.id
WHERE users.deleted_at IS NULL
GROUP BY users.id;

-- queryMany: Users2
SELECT *
FROM (
    SELECT * FROM users WHERE deleted_at IS NULL
) AS u
INNER JOIN items ON items.user_id = u.id;
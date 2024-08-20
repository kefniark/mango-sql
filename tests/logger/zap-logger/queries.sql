-- queryMany: UserNotDeleted
SELECT *
FROM users
WHERE users.deleted_at IS NULL;

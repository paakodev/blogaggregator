-- name: CreateUser :one
INSERT INTO USERS (id, name)
VALUES (
    $1,
    $2
)
RETURNING *;

-- name: GetUserById :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByName :one
SELECT *
FROM users
WHERE name = $1
LIMIT 1;

-- name: Reset :exec
DELETE FROM users;

-- name: GetAllUsers :many
SELECT *
FROM users;
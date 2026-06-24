-- name: CreateFeed :one
INSERT INTO feeds (id, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetFeedForUser :many
SELECT *
FROM feeds
WHERE user_id = $1;

-- name: GetAllFeeds :many
SELECT *
FROM feeds;

-- name: GetAllFeedsWithUsers :many
SELECT feeds.id, feeds.name, feeds.url, users.name AS user_name
FROM feeds
INNER JOIN users
  ON feeds.user_id = users.id;
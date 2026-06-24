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

-- name: GetFeedByURL :one
SELECT feeds.id, feeds.name, feeds.url
FROM feeds
WHERE feeds.url = $1;

-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, feed_id, user_id)
    VALUES ($1, $2, $3)
    RETURNING *
)
SELECT
    inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN feeds
    ON inserted_feed_follow.feed_id = feeds.id
INNER JOIN users
    ON inserted_feed_follow.user_id = users.id;

-- name: GetFeedFollowsForUser :many
SELECT 
    feed_follows.id, 
    feed_follows.feed_id, 
    feed_follows.user_id, 
    feeds.name AS feed_name, 
    users.name AS user_name
FROM feed_follows
INNER JOIN feeds
    ON feed_follows.feed_id = feeds.id
INNER JOIN users
    ON feed_follows.user_id = users.id
WHERE users.name = $1;
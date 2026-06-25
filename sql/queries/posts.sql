-- name: AddPost :one
INSERT INTO posts (id, title, url, description, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetPostsForUser :many
SELECT 
    posts.title,
    posts.url,
    posts.description,
    feeds.name AS feed_name
FROM posts
INNER JOIN feeds
    ON posts.feed_id = feeds.id
INNER JOIN feed_follows
    ON feeds.id = feed_follows.feed_id
INNER JOIN users
    ON feed_follows.user_id = users.id
WHERE users.name = $1
ORDER BY posts.created_at DESC
LIMIT $2;
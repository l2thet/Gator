-- name: CreateFeedFollow :one
INSERT INTO feed_follows (
    id,
    created_at,
    updated_at,
    feed_id,
    user_id
)
VALUES ($1, $2, $3, $4, $5)
RETURNING 
    id,
    created_at,
    updated_at,
    user_id,
    feed_id,
    (SELECT name FROM users WHERE id = user_id) as user_name,
    (SELECT name FROM feeds WHERE id = feed_id) as feed_name;
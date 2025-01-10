-- name: GetFeeds :many
SELECT
    f.*,
    u.name as user_name
FROM feeds f
JOIN users u ON f.user_id = u.id;
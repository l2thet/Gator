-- name: GetPostsForUser :many
SELECT 
    * 
FROM posts p
JOIN feeds f ON p.feed_id = f.id
WHERE f.user_id = $1
LIMIT $2;
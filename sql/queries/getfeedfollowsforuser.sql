-- name: GetFeedFollowsForUser :many
SELECT
    ff.*,
    u.name as user_name,
    f.name as feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f on ff.feed_id = f.id
WHERE ff.user_id = $1;
-- name: GetFeedstoFetch :many
SELECT
    *
FROM feeds
WHERE last_fetched_at IS NULL
ORDER BY updated_at DESC;
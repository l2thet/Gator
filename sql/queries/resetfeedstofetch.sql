-- name: ResetFeedsToFetch :exec
UPDATE feeds
    SET last_fetched_at = NULL
WHERE last_fetched_at IS NOT NULL;
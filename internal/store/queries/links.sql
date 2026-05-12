-- name: ListLinks :many
SELECT * FROM links
ORDER BY create_time DESC
LIMIT $1 OFFSET $2;

-- name: GetLink :one
SELECT * FROM links WHERE id = $1;

-- name: GetLinkByReallyLongPath :one
SELECT * FROM links WHERE really_long_path = $1;

-- name: GetLinkByOriginalUrl :one
SELECT * FROM links WHERE original_url = $1;

-- name: CreateLink :one
INSERT INTO links (original_url, really_long_path, really_long_query)
VALUES ($1, $2, $3)
RETURNING *;

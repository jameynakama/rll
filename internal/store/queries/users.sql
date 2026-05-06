-- name: ListLinks :many
SELECT * FROM links
ORDER BY create_time DESC
LIMIT $1 OFFSET $2;

-- name: GetLink :one
SELECT * FROM links WHERE id = $1;

-- name: CreateLink :one
INSERT INTO links (original_url, really_long_url)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateLink :one
UPDATE links
SET original_url = $2, really_long_url = $3
WHERE id = $1
RETURNING *;

-- name: DeleteLink :exec
DELETE FROM links WHERE id = $1;

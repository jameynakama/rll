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
INSERT INTO links (original_url, really_long_path, really_long_query, path_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListLinksSince :many
SELECT * FROM links WHERE create_time >= $1 ORDER BY id;

-- name: DeleteLink :exec
DELETE FROM links WHERE id = $1;

-- name: IsDomainBlocked :one
SELECT EXISTS (
    SELECT 1 FROM blocked_domains WHERE domain_matches(@host::text, domain)
);

-- name: BlockDomain :execrows
INSERT INTO blocked_domains (domain) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: DeleteLinksForDomain :execrows
DELETE FROM links WHERE domain_matches(url_host(original_url), @domain::text);

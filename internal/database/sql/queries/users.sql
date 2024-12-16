-- name: CreateUser :one
INSERT INTO users (
    email, api_key
) VALUES ( $1, encode(sha256(random()::text::bytea), 'hex'))
RETURNING *;


-- name: GetUserByApiKey :one
SELECT * FROM users
WHERE api_key = $1 LIMIT 1;


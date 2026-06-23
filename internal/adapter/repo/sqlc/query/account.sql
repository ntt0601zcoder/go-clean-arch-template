-- name: CreateAccount :exec
INSERT INTO accounts (
    id, email, first_name, last_name, password_hash, status, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: GetAccountByEmail :one
SELECT * FROM accounts WHERE email = $1;

-- name: UpdateAccount :execrows
UPDATE accounts SET
    first_name = COALESCE(sqlc.narg('first_name'), first_name),
    last_name  = COALESCE(sqlc.narg('last_name'), last_name),
    status     = COALESCE(sqlc.narg('status'), status),
    updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id');

-- name: DeleteAccount :execrows
DELETE FROM accounts WHERE id = $1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE (sqlc.narg('status')::int IS NULL OR status = sqlc.narg('status')::int)
ORDER BY created_at DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountAccounts :one
SELECT count(*) FROM accounts
WHERE (sqlc.narg('status')::int IS NULL OR status = sqlc.narg('status')::int);

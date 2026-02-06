-- Users SQL queries
-- Reference: specs/orchestrator.md §4.1

-- name: CreateUser :one
INSERT INTO users (
    github_id,
    github_username,
    github_token
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByGitHubID :one
SELECT * FROM users
WHERE github_id = $1 LIMIT 1;

-- name: UpdateUserToken :one
UPDATE users
SET github_token = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET github_username = $2,
    github_token = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

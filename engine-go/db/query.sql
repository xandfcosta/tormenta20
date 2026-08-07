-- Queries compiled by sqlc into db/sqlcgen. One camelCase column set means the
-- generated json tags already match the frontend contract (hpMax, catalogSpellId).
-- Grouped by domain; grows per Fase B slice.

-- users / auth (B.2)

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, name, passwordHash, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

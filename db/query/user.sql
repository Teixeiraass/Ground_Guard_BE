-- name: CreateUser :one
INSERT INTO users (
  username,
  hashed_password,
  full_name,
  email,
  profile_image
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE uuid = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUserName :one
UPDATE users
SET full_name = $2
WHERE uuid = $1
RETURNING *;

-- name: UpdateUserProfileImage :one
UPDATE users
SET profile_image = $2
WHERE uuid = $1
RETURNING *;
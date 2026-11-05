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
-- Case-insensitive so "Ana@x.com" and "ana@x.com" are the same account.
SELECT * FROM users
WHERE lower(email) = lower($1) LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

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

-- name: ListUsers :many
SELECT * FROM users
WHERE @search::text = '' OR username ILIKE '%' || @search::text || '%' OR email ILIKE '%' || @search::text || '%'
ORDER BY id
LIMIT @page_limit
OFFSET @page_offset;

-- name: BanUser :one
UPDATE users
SET status = 'BANNED', ban_reason = $2, banned_at = now()
WHERE uuid = $1
RETURNING *;

-- name: UnbanUser :one
UPDATE users
SET status = 'ACTIVE', ban_reason = NULL, banned_at = NULL
WHERE uuid = $1
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET hashed_password = $2, password_changed_at = now()
WHERE uuid = $1
RETURNING *;

-- name: SoftDeleteUser :one
UPDATE users
SET status = 'DELETED', deleted_at = now()
WHERE uuid = $1
RETURNING *;

-- name: UpdateUserDeviceInfo :one
UPDATE users
SET last_device_model = $2, last_platform = $3, last_os_version = $4, last_app_version = $5, last_seen_at = now()
WHERE uuid = $1
RETURNING *;

-- name: SetEmailVerificationToken :one
UPDATE users
SET email_verification_token = $2, email_verification_expires_at = $3
WHERE uuid = $1
RETURNING *;

-- name: VerifyUserEmailByToken :one
UPDATE users
SET email_verified = true, email_verification_token = NULL, email_verification_expires_at = NULL
WHERE email_verification_token = $1 AND email_verification_expires_at > now()
RETURNING *;

-- name: MarkUserEmailVerified :one
UPDATE users
SET email_verified = true, email_verification_token = NULL, email_verification_expires_at = NULL
WHERE uuid = $1
RETURNING *;
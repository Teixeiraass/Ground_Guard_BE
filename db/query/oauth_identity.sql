-- name: CreateOAuthIdentity :one
INSERT INTO oauth_identities (
  user_id,
  provider,
  provider_subject,
  email,
  email_verified
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetOAuthIdentityByProviderAndSubject :one
SELECT *
FROM oauth_identities
WHERE provider = $1 AND provider_subject = $2
LIMIT 1;
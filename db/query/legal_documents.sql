-- name: GetLegalDocument :one
SELECT *
FROM legal_documents
WHERE uuid = $1
AND active = true
LIMIT 1;

-- name: ListLegalDocuments :many
SELECT *
FROM legal_documents
WHERE active = true
ORDER BY published_at DESC
LIMIT $1
OFFSET $2;
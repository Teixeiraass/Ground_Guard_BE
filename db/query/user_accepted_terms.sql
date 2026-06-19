-- name: CreateUserAcceptedTerm :one
INSERT INTO user_accepted_terms (
  user_id, 
  legal_document_id
) VALUES (
  $1, $2
) RETURNING *;

-- name: GetUserAcceptedTerm :one
SELECT * FROM user_accepted_terms
WHERE uuid = $1 LIMIT 1;

-- name: GetUserAcceptedTermByUser :one
SELECT * FROM user_accepted_terms
WHERE user_id = $1 LIMIT 1;

-- name: GetUserAcceptedTermByLegalDocument :one
SELECT * FROM user_accepted_terms
WHERE legal_document_id = $1 LIMIT 1;
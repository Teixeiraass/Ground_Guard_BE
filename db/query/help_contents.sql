-- name: GetHelpContent :one
SELECT * FROM help_contents
WHERE uuid = $1 LIMIT 1;

-- name: ListHelpContents :many
SELECT *
FROM help_contents
WHERE published = true
ORDER BY order_number ASC
LIMIT $1
OFFSET $2;

-- name: ListHelpContentsByCategory :many
SELECT *
FROM help_contents
WHERE category = $1
AND published = true
ORDER BY order_number ASC
LIMIT $2
OFFSET $3;
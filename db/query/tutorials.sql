-- name: GetTutorial :one
SELECT *
FROM tutorials
WHERE uuid = $1
AND published = true
LIMIT 1;

-- name: ListTutorials :many
SELECT *
FROM tutorials
WHERE published = true
ORDER BY order_number ASC
LIMIT $1
OFFSET $2;

-- name: ListTutorialsByCategory :many
SELECT *
FROM tutorials
WHERE category = $1
AND published = true
ORDER BY order_number ASC
LIMIT $2
OFFSET $3;
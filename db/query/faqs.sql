-- name: GetFaq :one
SELECT * FROM faqs
WHERE uuid = $1 LIMIT 1;

-- name: ListFaqs :many
SELECT *
FROM faqs
WHERE published = true
ORDER BY order_number ASC
LIMIT $1
OFFSET $2;
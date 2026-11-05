-- name: CreateSupportMessage :one
INSERT INTO support_messages (
    user_id,
    sender,
    author_id,
    body
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSupportMessagesByUser :many
SELECT *
FROM support_messages
WHERE user_id = $1
ORDER BY id DESC
LIMIT $2
OFFSET $3;

-- name: MarkSupportMessagesRead :exec
UPDATE support_messages
SET read_at = now()
WHERE user_id = $1 AND sender = $2 AND read_at IS NULL;

-- name: ListSupportConversations :many
WITH last_message AS (
    SELECT DISTINCT ON (user_id) user_id, body, sender, created_at
    FROM support_messages
    ORDER BY user_id, id DESC
), unread AS (
    SELECT user_id, count(*) AS unread_count
    FROM support_messages
    WHERE sender = 'USER' AND read_at IS NULL
    GROUP BY user_id
)
SELECT
    u.uuid AS user_uuid,
    u.full_name,
    u.email,
    lm.body AS last_body,
    lm.sender AS last_sender,
    lm.created_at AS last_message_at,
    COALESCE(unread.unread_count, 0)::bigint AS unread_count
FROM last_message lm
JOIN users u ON u.id = lm.user_id
LEFT JOIN unread ON unread.user_id = lm.user_id
ORDER BY lm.created_at DESC
LIMIT $1
OFFSET $2;

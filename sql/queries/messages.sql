-- name: CreateMessage :one
insert into messages(
    id, description, sender_id, reciever_id,
    group_id, sent, recieved, created_at, updated_at
)
values(
    gen_random_uuid(),
    $1, $2, $3, $4, $5, $6, NOW(), NOW()
)
returning *;

-- name: UpdateMessage :one
update messages set description = $1 where id = $2 and sender_id = $3 and group_id = $4
returning description, sender_id, reciever_id, group_id sent, recieved, created_at, updated_at;

-- name: UpdateMessageRecieved :exec
update messages set recieved = true where id = $1;

-- name: RemoveMessage :exec
delete from messages where id = $1 and sender_id = $2 and reciever_id = $3 and group_id = $4;

-- name: GetAllMessages :many
select * from messages where sender_id = $1 and reciever_id = $2;

-- name: GetAllGroupMessages :many
select * from messages where group_id = $1;

-- name: GetLatestMessagesByRecieverID :many
select users.username as sender, messages.description as messages, count(*) as total_new_messages
from messages join users on messages.sender_id = users.id where messages.reciever_id = $1 and
messages.created_at > $2 group by users.username order by messages.created_at;

-- name: GetLatestGroupMessagesByGroupID :many
select groups.name as group_name, messages.description as messages, count(*) as total_new_messages
from messages join groups on messages.group_id = groups.id where messages.created_at > $1 group by group_name
order by messages.created_at;
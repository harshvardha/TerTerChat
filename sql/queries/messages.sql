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
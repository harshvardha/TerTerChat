-- name: CreateMessage :one
insert into messages(
    id, description, sender_id, reciever_id,
    group_id, sent, created_at, updated_at
)
values(
    gen_random_uuid(),
    $1, $2, $3, $4, $5, NOW(), NOW()
)
returning *;

-- name: UpdateMessage :one
update messages set description = $1, updated_at = NOW() where id = $2 and sender_id = $3 and group_id = $4
returning description, sender_id, reciever_id, group_id, sent, recieved, read, is_receiver_allowed_to_see, created_at, updated_at;

-- name: GetMessageSenderReceiverAndGroupID :one
select sender_id, reciever_id, group_id from messages where id = $1;

-- name: MarkIsSenderAllowedToSeeFalse :exec
update messages set is_sender_allowed_to_see = false where sender_id = $1 and reciever_id = $2;

-- name: MarkIsReceiverAllowedToSeeFalse :exec
update messages set is_receiver_allowed_to_see = false where sender_id = $1 and reciever_id = $2;

-- name: GetAllMessages :many
select * from messages where sender_id = $1 and reciever_id = $2 and created_at < $3 order by created_at limit 10;

-- name: GetAllGroupMessages :many
select * from messages where group_id = $1 and created_at < $2 order by created_at limit 10;

-- name: GetLatestMessagesByRecieverID :many
select users.username as sender, messages.description as messages, count(*) as total_new_messages
from messages join users on messages.sender_id = users.id where messages.reciever_id = $1 and
messages.created_at > $2 group by users.username order by messages.created_at;

-- name: GetLatestGroupMessagesByGroupID :many
select groups.name as group_name, messages.description as messages, count(*) as total_new_messages
from messages join groups on messages.group_id = groups.id where messages.created_at > $1 group by group_name
order by messages.created_at;

-- name: MarkMessageReceived :one
update messages set received = true and updated_at = NOW() where id = $1
returning updated_at;

-- name: MarkMessageRead :one
update messages set read = true and updated_at = NOW() where id = $1
returning updated_at;

-- name: MarkGroupMessageRead :exec
insert into group_message_read(message_id, group_member_id, group_id, read_at)
values($1, $2, $3, NOW());

-- name: MarkGroupMessageReceived :exec
insert into group_message_received(message_id, group_member_id, group_id, received_at)
values($1, $2, $3, NOW());

-- name: CountOfGroupMembersWhoReceivedMessage :one
select count(*) from group_message_received where message_id = $1 and group_member_id = $2 and group_id = $3;

-- name: CountOfGroupMembersWhoReadMessage :one
select count(*) from group_message_read where message_id = $1 and group_member_id = $2 and group_id = $3;

-- name: GetAllOneToOneConversations :many
select distinct messages.reciever_id as reciever_id, users.username as username from messages join users on messages.reciever_id = users.id where messages.sender_id = $1;

-- name: GetAllGroupConversations :many
select distinct messages.group_id as group_id, groups.name as group_name from messages join groups on messages.group_id = groups.id where messages.sender_id = $1;

-- name: AddReceiverToGroupMessage :exec
insert into group_message_receivers(
    message_id, member_id, group_id
) values(
    $1, $2, $3
);

-- name: MarkIsAllowedToSeeAsFalseForGroupMemeberReceivers :exec
update group_message_receivers set is_allowed_to_see = false where message_id = $1 and group_id = $2;

-- name: MarkIsAllowedToSeeAsFalseForSpecificGroupMemeber :exec
update group_message_receivers set is_allowed_to_see = false where message_id = $1 and group_id = $2 and member_id = $3;

-- name: MarkIsAllowedToSeeAsFalseForSpecificGroupMemeberWhoLeaves :exec
update group_message_receivers set is_allowed_to_see = false where group_id = $1 and member_id = $2;

-- name: MarkIsAllowedToSeeAsTrueForSpecificGroupMember :exec
update group_message_receivers set is_allowed_to_see = true where group_id = $1 and member_id = $2;

-- name: IsGroupMemberAllowedToSeeMessage :one
select is_allowed_to_see from group_message_receivers where message_id = $1 and group_id = $2 and member_id = $3;
-- name: CreateGroup :one
insert into groups(id, name, created_at, updated_at)
values(gen_random_uuid(), $1, NOW(), NOW())
returning *;

-- name: UpdateGroup :one
update groups set name = $1 where id = $2
returning name, updated_at;

-- name: AddUserToGroup :exec
insert into users_groups(user_id, group_id, created_at)
values($1, $2, NOW());

-- name: RemoveUserFromGroup :exec
delete from users_groups where user_id = $1 and group_id = $2;

-- name: MakeUserAdmin :exec
insert into group_admins(user_id, group_id, created_at)
values($1, $2, NOW());

-- name: RemoveUserFromAdmin :exec
delete from group_admins where user_id = $1 and group_id = $2;

-- name: GetGroupMembers :many
select groups.name, users.username from groups join users_groups on groups.id = users_groups.group_id join users on users.id = users_groups.user_id where groups.id = $1;

-- name: GetGroupAdmins :many
select groups.name, users.username from groups join group_admins on groups.id = group_admins.group_id join users on users.id = group_admins.user_id where groups.id = $1;

-- name: DeleteGroup :exec
delete from groups where id = $1;
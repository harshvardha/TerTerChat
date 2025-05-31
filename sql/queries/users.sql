-- name: CreateUser :one
insert into users(id, email, username, password, created_at, updated_at)
values(
    gen_random_uuid(), $1, $2, $3, NOW(), NOW()
)
returning id, email, username, created_at, updated_at;

-- name: UpdateEmail :exec
update users set email = $1 where id = $2;

-- name: UpdatePassword :exec
update users set password = $1 where id = $2;

-- name: UpdateUsername :one
update users set username = $1 where id = $2
returning username, updated_at;

-- name: GetUserByEmail :one
select id, username, password from users where email = $1;

-- name: GetUserById :one
select email, username, created_at, updated_at from users where id = $1;

-- name: RemoveUser :exec
delete from users where id = $1;
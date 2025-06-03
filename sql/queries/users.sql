-- name: CreateUser :one
insert into users(id, phonenumber, username, password, created_at, updated_at)
values(
    gen_random_uuid(), $1, $2, $3, NOW(), NOW()
)
returning id, phonenumber, username, created_at, updated_at;

-- name: UpdatePhonenumber :exec
update users set phonenumber = $1 where id = $2;

-- name: UpdatePassword :exec
update users set password = $1 where id = $2;

-- name: UpdateUsername :one
update users set username = $1 where id = $2
returning username, updated_at;

-- name: GetUserByPhonenumber :one
select id, username, password from users where phonenumber = $1;

-- name: GetUserById :one
select phonenumber, username, created_at, updated_at from users where id = $1;

-- name: DoesUserExist :one
select 1 from users where phonenumber = $1;

-- name: RemoveUser :exec
delete from users where id = $1;
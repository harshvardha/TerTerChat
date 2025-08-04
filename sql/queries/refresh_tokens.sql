-- name: CreateRefreshToken :exec
insert into refresh_token(token, user_id, created_at, expires_at)
values($1, $2, NOW(), $3);

-- name: GetRefreshTokenExpirationTime :one
select expires_at from refresh_token where user_id = $1;

-- name: RemoveRefreshToken :exec
delete from refresh_token where user_id = $1;
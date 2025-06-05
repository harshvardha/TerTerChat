-- +goose Up
create table refresh_token(
    token text not null unique,
    user_id uuid unique not null references users(id) on delete cascade,
    created_at timestamp not null,
    expires_at timestamp not null
);

-- +goose Down
drop table refresh_token;
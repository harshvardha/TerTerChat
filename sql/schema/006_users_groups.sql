-- +goose Up
create table users_groups(
    user_id uuid not null references users(id) on delete cascade,
    group_id uuid not null references groups(id) on delete cascade,
    created_at timestamp not null,
    unique(user_id, group_id)
);

-- +goose Down
drop table users_groups;
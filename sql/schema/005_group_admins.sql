-- +goose Up
create table group_admins(
    group_id uuid not null references groups(id) on delete cascade,
    user_id uuid not null references users(id),
    created_at timestamp not null,
    unique(group_id, user_id)
);

-- +goose Down
drop table group_admins;
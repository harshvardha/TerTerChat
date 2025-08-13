-- +goose Up
create table group_message_read(
    message_id uuid not null references messages(id) on delete cascade,
    group_member_id uuid not null references users(id) on delete cascade,
    group_id uuid not null references groups(id) on delete cascade,
    created_at timestamp not null,
    unique(message_id, group_member_id, group_id)
);

-- +goose Down
drop table group_message_read;
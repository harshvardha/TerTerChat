-- +goose Up
create table group_message_receivers(
    message_id uuid not null references messages(id) on delete cascade,
    member_id uuid not null references users(id) on delete cascade,
    group_id uuid not null references groups(id) on delete cascade,
    is_allowed_to_see boolean not null default true
);

-- +goose Down
drop table group_message_receivers;
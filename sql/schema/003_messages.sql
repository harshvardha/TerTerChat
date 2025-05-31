-- +goose Up
create table messages(
    id uuid not null primary key,
    description text not null,
    sender_id uuid not null references users(id) on delete cascade,
    reciever_id uuid references users(id),
    group_id uuid references groups(id),
    sent boolean not null default false,
    recieved boolean not null default false,
    created_at timestamp not null,
    updated_at timestamp not null
);

-- +goose Down
drop table messages;
-- +goose Up
create table groups(
    id uuid not null primary key,
    name varchar(100) not null,
    created_at timestamp not null,
    updated_at timestamp not null
);

-- +goose Down
drop table groups;
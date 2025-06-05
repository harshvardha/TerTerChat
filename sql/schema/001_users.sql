-- +goose Up
create table users(
    id uuid not null primary key,
    phonenumber varchar(13) not null unique,
    username varchar(50) not null,
    password text not null,
    last_available timestamp,
    created_at timestamp not null,
    updated_at timestamp not null
);

-- +goose Down
drop table users;
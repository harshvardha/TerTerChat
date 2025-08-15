-- +goose Up
alter table messages add column is_sender_allowed_to_see boolean not null default true,
add column is_receiver_allowed_to_see boolean not null default true;

-- +goose Down
alter table drop column is_sender_allowed_to_see, drop column is_receiver_allowed_to_see;
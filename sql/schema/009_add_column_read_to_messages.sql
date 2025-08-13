-- +goose Up
alter table messages add column read bool not null default false;

-- +goose Down
alter table drop column read;
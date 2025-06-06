// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: users.sql

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const createUser = `-- name: CreateUser :one
insert into users(id, phonenumber, username, password, created_at, updated_at)
values(
    gen_random_uuid(), $1, $2, $3, NOW(), NOW()
)
returning id, phonenumber, username, created_at, updated_at
`

type CreateUserParams struct {
	Phonenumber string
	Username    string
	Password    string
}

type CreateUserRow struct {
	ID          uuid.UUID
	Phonenumber string
	Username    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Phonenumber, arg.Username, arg.Password)
	var i CreateUserRow
	err := row.Scan(
		&i.ID,
		&i.Phonenumber,
		&i.Username,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const doesUserExist = `-- name: DoesUserExist :one
select 1 from users where phonenumber = $1
`

func (q *Queries) DoesUserExist(ctx context.Context, phonenumber string) (int32, error) {
	row := q.db.QueryRowContext(ctx, doesUserExist, phonenumber)
	var column_1 int32
	err := row.Scan(&column_1)
	return column_1, err
}

const getUserById = `-- name: GetUserById :one
select phonenumber, username, created_at, updated_at from users where id = $1
`

type GetUserByIdRow struct {
	Phonenumber string
	Username    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (q *Queries) GetUserById(ctx context.Context, id uuid.UUID) (GetUserByIdRow, error) {
	row := q.db.QueryRowContext(ctx, getUserById, id)
	var i GetUserByIdRow
	err := row.Scan(
		&i.Phonenumber,
		&i.Username,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByPhonenumber = `-- name: GetUserByPhonenumber :one
select id, username, password, last_available, created_at from users where phonenumber = $1
`

type GetUserByPhonenumberRow struct {
	ID            uuid.UUID
	Username      string
	Password      string
	LastAvailable sql.NullTime
	CreatedAt     time.Time
}

func (q *Queries) GetUserByPhonenumber(ctx context.Context, phonenumber string) (GetUserByPhonenumberRow, error) {
	row := q.db.QueryRowContext(ctx, getUserByPhonenumber, phonenumber)
	var i GetUserByPhonenumberRow
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Password,
		&i.LastAvailable,
		&i.CreatedAt,
	)
	return i, err
}

const getUserPhonenumberByID = `-- name: GetUserPhonenumberByID :one
select phonenumber from users where id = $1
`

func (q *Queries) GetUserPhonenumberByID(ctx context.Context, id uuid.UUID) (string, error) {
	row := q.db.QueryRowContext(ctx, getUserPhonenumberByID, id)
	var phonenumber string
	err := row.Scan(&phonenumber)
	return phonenumber, err
}

const removeUser = `-- name: RemoveUser :exec
delete from users where id = $1
`

func (q *Queries) RemoveUser(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, removeUser, id)
	return err
}

const setLastAvailable = `-- name: SetLastAvailable :exec
update users set last_available = NOW() where phonenumber = $1
`

func (q *Queries) SetLastAvailable(ctx context.Context, phonenumber string) error {
	_, err := q.db.ExecContext(ctx, setLastAvailable, phonenumber)
	return err
}

const updatePassword = `-- name: UpdatePassword :exec
update users set password = $1 where id = $2
`

type UpdatePasswordParams struct {
	Password string
	ID       uuid.UUID
}

func (q *Queries) UpdatePassword(ctx context.Context, arg UpdatePasswordParams) error {
	_, err := q.db.ExecContext(ctx, updatePassword, arg.Password, arg.ID)
	return err
}

const updatePhonenumber = `-- name: UpdatePhonenumber :exec
update users set phonenumber = $1 where id = $2
`

type UpdatePhonenumberParams struct {
	Phonenumber string
	ID          uuid.UUID
}

func (q *Queries) UpdatePhonenumber(ctx context.Context, arg UpdatePhonenumberParams) error {
	_, err := q.db.ExecContext(ctx, updatePhonenumber, arg.Phonenumber, arg.ID)
	return err
}

const updateUsername = `-- name: UpdateUsername :one
update users set username = $1 where id = $2
returning username
`

type UpdateUsernameParams struct {
	Username string
	ID       uuid.UUID
}

func (q *Queries) UpdateUsername(ctx context.Context, arg UpdateUsernameParams) (string, error) {
	row := q.db.QueryRowContext(ctx, updateUsername, arg.Username, arg.ID)
	var username string
	err := row.Scan(&username)
	return username, err
}

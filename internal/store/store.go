package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrNotFound   = errors.New("not found")
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const userColumns = `id, email, role, created_at, password_hash`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING `+userColumns,
		email, passwordHash))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, ErrEmailTaken
	}
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`, email))
}

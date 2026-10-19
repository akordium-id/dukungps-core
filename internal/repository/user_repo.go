package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/user"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByID(ctx context.Context, id int64) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindAll(ctx context.Context) ([]*user.User, error)
	Update(ctx context.Context, u *user.User) error
	Delete(ctx context.Context, id int64) error
}

type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

func (r *PostgresUserRepo) Create(ctx context.Context, u *user.User) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		u.Name, u.Email, u.Password, u.Role,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	return err
}

func (r *PostgresUserRepo) FindByID(ctx context.Context, id int64) (*user.User, error) {
	u := &user.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, email, password, role, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	u := &user.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, email, password, role, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresUserRepo) FindAll(ctx context.Context) ([]*user.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, email, password, role, created_at, updated_at
		 FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*user.User, 0)
	for rows.Next() {
		u := &user.User{}
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *PostgresUserRepo) Update(ctx context.Context, u *user.User) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users
		 SET name = $1, email = $2, password = $3, role = $4, updated_at = NOW()
		 WHERE id = $5`,
		u.Name, u.Email, u.Password, u.Role, u.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

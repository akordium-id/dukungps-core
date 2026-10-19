package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/command"
)

var ErrCommandNotFound = errors.New("command not found")

type CommandRepository interface {
	Create(ctx context.Context, c *command.Command) error
	FindByID(ctx context.Context, id int64) (*command.Command, error)
	FindByDeviceID(ctx context.Context, deviceID int64) ([]command.Command, error)
	FindAll(ctx context.Context) ([]command.Command, error)
	Update(ctx context.Context, c *command.Command) error
	Delete(ctx context.Context, id int64) error
	MarkSent(ctx context.Context, id int64, sentAt time.Time) error
	MarkFailed(ctx context.Context, id int64) error
}

type PostgresCommandRepo struct {
	pool *pgxpool.Pool
}

func NewCommandRepo(pool *pgxpool.Pool) *PostgresCommandRepo {
	return &PostgresCommandRepo{pool: pool}
}

func (r *PostgresCommandRepo) Create(ctx context.Context, c *command.Command) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO commands (device_id, type, data, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, sent_at, created_at`,
		c.DeviceID, c.Type, c.Data, c.Status,
	).Scan(&c.ID, &c.SentAt, &c.CreatedAt)
	return err
}

func (r *PostgresCommandRepo) FindByID(ctx context.Context, id int64) (*command.Command, error) {
	c := &command.Command{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, type, data, status, sent_at, created_at
		 FROM commands WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.DeviceID, &c.Type, &c.Data, &c.Status, &c.SentAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCommandNotFound
	}
	return c, err
}

func (r *PostgresCommandRepo) FindByDeviceID(ctx context.Context, deviceID int64) ([]command.Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, data, status, sent_at, created_at
		 FROM commands WHERE device_id = $1 ORDER BY created_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := make([]command.Command, 0)
	for rows.Next() {
		var c command.Command
		if err := rows.Scan(
			&c.ID, &c.DeviceID, &c.Type, &c.Data, &c.Status, &c.SentAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (r *PostgresCommandRepo) FindAll(ctx context.Context) ([]command.Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, data, status, sent_at, created_at
		 FROM commands ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := make([]command.Command, 0)
	for rows.Next() {
		var c command.Command
		if err := rows.Scan(
			&c.ID, &c.DeviceID, &c.Type, &c.Data, &c.Status, &c.SentAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (r *PostgresCommandRepo) Update(ctx context.Context, c *command.Command) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE commands
		 SET device_id = $1, type = $2, data = $3, status = $4, sent_at = $5
		 WHERE id = $6`,
		c.DeviceID, c.Type, c.Data, c.Status, c.SentAt, c.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

func (r *PostgresCommandRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM commands WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

func (r *PostgresCommandRepo) FindPendingByDeviceID(ctx context.Context, deviceID int64) ([]command.Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, data, status, sent_at, created_at
		 FROM commands WHERE device_id = $1 AND status = $2 ORDER BY created_at ASC`,
		deviceID, command.StatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := make([]command.Command, 0)
	for rows.Next() {
		var c command.Command
		if err := rows.Scan(
			&c.ID, &c.DeviceID, &c.Type, &c.Data, &c.Status, &c.SentAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (r *PostgresCommandRepo) MarkSent(ctx context.Context, id int64, sentAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE commands SET status = $1, sent_at = $2 WHERE id = $3`,
		command.StatusSent, sentAt, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

func (r *PostgresCommandRepo) MarkFailed(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE commands SET status = $1 WHERE id = $2`,
		command.StatusFailed, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

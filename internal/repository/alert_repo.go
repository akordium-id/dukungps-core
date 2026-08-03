package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/alert"
)

var ErrAlertNotFound = errors.New("alert not found")

type AlertRepository interface {
	Create(ctx context.Context, a *alert.Alert) error
	FindByID(ctx context.Context, id int64) (*alert.Alert, error)
	FindAll(ctx context.Context) ([]alert.Alert, error)
	FindByDeviceID(ctx context.Context, deviceID int64) ([]alert.Alert, error)
	FindPending(ctx context.Context) ([]alert.Alert, error)
	Update(ctx context.Context, a *alert.Alert) error
	Delete(ctx context.Context, id int64) error
}

type PostgresAlertRepo struct {
	pool *pgxpool.Pool
}

func NewAlertRepo(pool *pgxpool.Pool) *PostgresAlertRepo {
	return &PostgresAlertRepo{pool: pool}
}

func (r *PostgresAlertRepo) Create(ctx context.Context, a *alert.Alert) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO alerts (device_id, type, title, description, status, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, sent_at, created_at`,
		a.DeviceID, a.Type, a.Title, a.Description, a.Status, a.Metadata,
	).Scan(&a.ID, &a.SentAt, &a.CreatedAt)
	return err
}

func (r *PostgresAlertRepo) FindByID(ctx context.Context, id int64) (*alert.Alert, error) {
	a := &alert.Alert{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, type, title, description, status, metadata, sent_at, created_at
		 FROM alerts WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.DeviceID, &a.Type, &a.Title, &a.Description, &a.Status,
		&a.Metadata, &a.SentAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAlertNotFound
	}
	return a, err
}

func (r *PostgresAlertRepo) FindByDeviceID(ctx context.Context, deviceID int64) ([]alert.Alert, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, title, description, status, metadata, sent_at, created_at
		 FROM alerts WHERE device_id = $1 ORDER BY created_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]alert.Alert, 0)
	for rows.Next() {
		var a alert.Alert
		var meta []byte
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.Type, &a.Title, &a.Description, &a.Status,
			&meta, &a.SentAt, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &a.Metadata); err != nil {
				return nil, err
			}
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *PostgresAlertRepo) FindPending(ctx context.Context) ([]alert.Alert, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, title, description, status, metadata, sent_at, created_at
		 FROM alerts WHERE status = $1 ORDER BY created_at ASC`,
		alert.AlertStatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]alert.Alert, 0)
	for rows.Next() {
		var a alert.Alert
		var meta []byte
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.Type, &a.Title, &a.Description, &a.Status,
			&meta, &a.SentAt, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &a.Metadata); err != nil {
				return nil, err
			}
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *PostgresAlertRepo) Update(ctx context.Context, a *alert.Alert) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE alerts
		 SET device_id = $1, type = $2, title = $3, description = $4, status = $5, metadata = $6, sent_at = $7
		 WHERE id = $8`,
		a.DeviceID, a.Type, a.Title, a.Description, a.Status, a.Metadata, a.SentAt, a.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (r *PostgresAlertRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM alerts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (r *PostgresAlertRepo) FindAll(ctx context.Context) ([]alert.Alert, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, type, title, description, status, metadata, sent_at, created_at
		 FROM alerts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]alert.Alert, 0)
	for rows.Next() {
		var a alert.Alert
		var meta []byte
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.Type, &a.Title, &a.Description, &a.Status,
			&meta, &a.SentAt, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &a.Metadata); err != nil {
				return nil, err
			}
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/device"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrDeviceExists   = errors.New("device already exists")
)

type DeviceRepository interface {
	Create(ctx context.Context, d *device.Device) error
	FindByUniqueID(ctx context.Context, uniqueID string) (*device.Device, error)
	Update(ctx context.Context, d *device.Device) error
}

type PostgresDeviceRepo struct {
	pool *pgxpool.Pool
}

func NewDeviceRepo(pool *pgxpool.Pool) *PostgresDeviceRepo {
	return &PostgresDeviceRepo{pool: pool}
}

func (r *PostgresDeviceRepo) Create(ctx context.Context, d *device.Device) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO devices (unique_id, name, model, protocol, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		d.UniqueID, d.Name, d.Model, d.Protocol, d.Status,
	).Scan(&d.ID, &d.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDeviceExists
	}
	return err
}

func (r *PostgresDeviceRepo) FindByUniqueID(ctx context.Context, uniqueID string) (*device.Device, error) {
	d := &device.Device{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, unique_id, name, model, protocol, status, created_at, updated_at
		 FROM devices WHERE unique_id = $1`,
		uniqueID,
	).Scan(&d.ID, &d.UniqueID, &d.Name, &d.Model, &d.Protocol, &d.Status, &d.CreatedAt, &d.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *PostgresDeviceRepo) Update(ctx context.Context, d *device.Device) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE devices
		 SET name = $1, model = $2, protocol = $3, status = $4, updated_at = NOW()
		 WHERE id = $5`,
		d.Name, d.Model, d.Protocol, d.Status, d.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

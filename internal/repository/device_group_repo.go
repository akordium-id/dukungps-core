package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/device"
)

var ErrDeviceGroupNotFound = errors.New("device group not found")

type DeviceGroupRepository interface {
	Create(ctx context.Context, g *device.DeviceGroup) error
	FindByID(ctx context.Context, id int64) (*device.DeviceGroup, error)
	FindAll(ctx context.Context) ([]*device.DeviceGroup, error)
	Update(ctx context.Context, g *device.DeviceGroup) error
	Delete(ctx context.Context, id int64) error
	AddDevice(ctx context.Context, groupID int64, deviceID int64) error
	RemoveDevice(ctx context.Context, groupID int64, deviceID int64) error
	FindDevicesByGroup(ctx context.Context, groupID int64) ([]*device.Device, error)
}

type PostgresDeviceGroupRepo struct {
	pool *pgxpool.Pool
}

func NewDeviceGroupRepo(pool *pgxpool.Pool) *PostgresDeviceGroupRepo {
	return &PostgresDeviceGroupRepo{pool: pool}
}

func (r *PostgresDeviceGroupRepo) Create(ctx context.Context, g *device.DeviceGroup) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO device_groups (name, description)
		 VALUES ($1, $2)
		 RETURNING id, created_at, updated_at`,
		g.Name, g.Description,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	return err
}

func (r *PostgresDeviceGroupRepo) FindByID(ctx context.Context, id int64) (*device.DeviceGroup, error) {
	g := &device.DeviceGroup{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM device_groups WHERE id = $1`,
		id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeviceGroupNotFound
	}
	return g, err
}

func (r *PostgresDeviceGroupRepo) FindAll(ctx context.Context) ([]*device.DeviceGroup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM device_groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]*device.DeviceGroup, 0)
	for rows.Next() {
		g := &device.DeviceGroup{}
		if err := rows.Scan(
			&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *PostgresDeviceGroupRepo) Update(ctx context.Context, g *device.DeviceGroup) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE device_groups
		 SET name = $1, description = $2, updated_at = NOW()
		 WHERE id = $3`,
		g.Name, g.Description, g.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceGroupNotFound
	}
	return nil
}

func (r *PostgresDeviceGroupRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM device_groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceGroupNotFound
	}
	return nil
}

func (r *PostgresDeviceGroupRepo) AddDevice(ctx context.Context, groupID int64, deviceID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_group_members (device_group_id, device_id)
		 VALUES ($1, $2)`,
		groupID, deviceID,
	)
	return err
}

func (r *PostgresDeviceGroupRepo) RemoveDevice(ctx context.Context, groupID int64, deviceID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM device_group_members
		 WHERE device_group_id = $1 AND device_id = $2`,
		groupID, deviceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("membership not found")
	}
	return nil
}

func (r *PostgresDeviceGroupRepo) FindDevicesByGroup(ctx context.Context, groupID int64) ([]*device.Device, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT d.id, d.unique_id, d.name, d.model, d.protocol, d.status, d.created_at, d.updated_at
		 FROM devices d
		 JOIN device_group_members m ON d.id = m.device_id
		 WHERE m.device_group_id = $1
		 ORDER BY d.id`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]*device.Device, 0)
	for rows.Next() {
		d := &device.Device{}
		if err := rows.Scan(
			&d.ID, &d.UniqueID, &d.Name, &d.Model, &d.Protocol, &d.Status, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}
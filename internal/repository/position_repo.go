package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

type PositionRepository interface {
	Create(ctx context.Context, p *position.Position) error
	FindByDeviceID(ctx context.Context, deviceID int64, limit int) ([]position.Position, error)
	Search(ctx context.Context, params PositionSearchParams) ([]position.Position, error)
}

type PositionSearchParams struct {
	DeviceID int64
	Limit    int
	From     *time.Time
	To       *time.Time
	MinSpeed float64
	MaxSpeed float64
}

type PostgresPositionRepo struct {
	pool *pgxpool.Pool
}

func NewPositionRepo(pool *pgxpool.Pool) *PostgresPositionRepo {
	return &PostgresPositionRepo{pool: pool}
}

func (r *PostgresPositionRepo) Create(ctx context.Context, p *position.Position) error {
	attributes, err := json.Marshal(p.Attributes)
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx,
		`INSERT INTO positions (device_id, latitude, longitude, altitude, speed, course, valid, time, device_time, attributes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at`,
		p.DeviceID, p.Latitude, p.Longitude, p.Altitude, p.Speed, p.Course, p.Valid,
		p.Time, p.DeviceTime, attributes,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *PostgresPositionRepo) FindByDeviceID(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, latitude, longitude, altitude, speed, course, valid, time, device_time, attributes
		 FROM positions
		 WHERE device_id = $1
		 ORDER BY time DESC
		 LIMIT $2`,
		deviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]position.Position, 0)
	for rows.Next() {
		var p position.Position
		var attrs []byte
		if err := rows.Scan(
			&p.ID, &p.DeviceID, &p.Latitude, &p.Longitude, &p.Altitude,
			&p.Speed, &p.Course, &p.Valid, &p.Time, &p.DeviceTime, &attrs,
		); err != nil {
			return nil, err
		}
		if len(attrs) > 0 {
			if err := json.Unmarshal(attrs, &p.Attributes); err != nil {
				return nil, err
			}
		}
		positions = append(positions, p)
	}

	return positions, rows.Err()
}

func (r *PostgresPositionRepo) Search(ctx context.Context, params PositionSearchParams) ([]position.Position, error) {
	query := `SELECT id, device_id, latitude, longitude, altitude, speed, course, valid, time, device_time, attributes
		 FROM positions WHERE device_id = $1`
	args := []any{params.DeviceID}
	argIdx := 1

	if params.From != nil {
		argIdx++
		query += ` AND time >= $` + strconv.Itoa(argIdx)
		args = append(args, *params.From)
	}

	if params.To != nil {
		argIdx++
		query += ` AND time <= $` + strconv.Itoa(argIdx)
		args = append(args, *params.To)
	}

	if params.MinSpeed >= 0 {
		argIdx++
		query += ` AND speed >= $` + strconv.Itoa(argIdx)
		args = append(args, params.MinSpeed)
	}

	if params.MaxSpeed >= 0 {
		argIdx++
		query += ` AND speed <= $` + strconv.Itoa(argIdx)
		args = append(args, params.MaxSpeed)
	}

	argIdx++
	query += ` ORDER BY time DESC LIMIT $` + strconv.Itoa(argIdx)
	args = append(args, params.Limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]position.Position, 0)
	for rows.Next() {
		var p position.Position
		var attrs []byte
		if err := rows.Scan(
			&p.ID, &p.DeviceID, &p.Latitude, &p.Longitude, &p.Altitude,
			&p.Speed, &p.Course, &p.Valid, &p.Time, &p.DeviceTime, &attrs,
		); err != nil {
			return nil, err
		}
		if len(attrs) > 0 {
			if err := json.Unmarshal(attrs, &p.Attributes); err != nil {
				return nil, err
			}
		}
		positions = append(positions, p)
	}

	return positions, rows.Err()
}

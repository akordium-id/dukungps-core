package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/geofence"
)

var ErrGeofenceNotFound = errors.New("geofence not found")

type GeofenceRepository interface {
	Create(ctx context.Context, g *geofence.Geofence) error
	FindByID(ctx context.Context, id int64) (*geofence.Geofence, error)
	FindAll(ctx context.Context) ([]*geofence.Geofence, error)
	Update(ctx context.Context, g *geofence.Geofence) error
	Delete(ctx context.Context, id int64) error
}

type PostgresGeofenceRepo struct {
	pool *pgxpool.Pool
}

func NewGeofenceRepo(pool *pgxpool.Pool) *PostgresGeofenceRepo {
	return &PostgresGeofenceRepo{pool: pool}
}

func (r *PostgresGeofenceRepo) Create(ctx context.Context, g *geofence.Geofence) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO geofences (name, description, type, area, center_lat, center_lon, radius)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		g.Name, g.Description, g.Type, g.Area, g.Center.Latitude, g.Center.Longitude, g.Radius,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	return err
}

func (r *PostgresGeofenceRepo) FindByID(ctx context.Context, id int64) (*geofence.Geofence, error) {
	g := &geofence.Geofence{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, type, area, center_lat, center_lon, radius, created_at, updated_at
		 FROM geofences WHERE id = $1`,
		id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.Type, &g.Area,
		&g.Center.Latitude, &g.Center.Longitude, &g.Radius, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGeofenceNotFound
	}
	return g, err
}

func (r *PostgresGeofenceRepo) FindAll(ctx context.Context) ([]*geofence.Geofence, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, type, area, center_lat, center_lon, radius, created_at, updated_at
		 FROM geofences ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	geofences := make([]*geofence.Geofence, 0)
	for rows.Next() {
		g := &geofence.Geofence{}
		if err := rows.Scan(
			&g.ID, &g.Name, &g.Description, &g.Type, &g.Area,
			&g.Center.Latitude, &g.Center.Longitude, &g.Radius, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		geofences = append(geofences, g)
	}
	return geofences, rows.Err()
}

func (r *PostgresGeofenceRepo) Update(ctx context.Context, g *geofence.Geofence) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE geofences
		 SET name = $1, description = $2, type = $3, area = $4,
		     center_lat = $5, center_lon = $6, radius = $7, updated_at = NOW()
		 WHERE id = $8`,
		g.Name, g.Description, g.Type, g.Area,
		g.Center.Latitude, g.Center.Longitude, g.Radius, g.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGeofenceNotFound
	}
	return nil
}

func (r *PostgresGeofenceRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM geofences WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGeofenceNotFound
	}
	return nil
}

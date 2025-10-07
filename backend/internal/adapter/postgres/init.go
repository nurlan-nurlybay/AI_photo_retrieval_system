package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
)

func InitDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pgxCfg.MaxConns = int32(cfg.Postgres.MaxOpenConns)
	pgxCfg.MinConns = int32(cfg.Postgres.MaxIdleConns)
	pgxCfg.MaxConnLifetime = cfg.Postgres.ConnMaxLifetime
	pgxCfg.MaxConnIdleTime = cfg.Postgres.ConnMaxIdleTime
	pgxCfg.HealthCheckPeriod = cfg.Postgres.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w ", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}

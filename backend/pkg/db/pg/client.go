package pg

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
	"github.com/pkg/errors"
)

type pgClient struct {
	masterDBC db.DB
}

func New(ctx context.Context, dsn string) (db.Client, error) {
	pgxCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Errorf("parse config: %v", err)
	}

	dbc, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, errors.Errorf("connect to db: %v", err)
	}

	return &pgClient{masterDBC: &pg{dbc: dbc}}, nil
}

func (c *pgClient) DB() db.DB {
	return c.masterDBC
}

func (c *pgClient) Close() error {
	if c.masterDBC != nil {
		c.masterDBC.Close()
	}

	return nil
}

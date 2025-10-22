package app

import (
	"context"
	"log"
	"log/slog"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/http"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/closer"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db/pg"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db/prettier"
)

type serviceProvider struct {
	ctx context.Context
	cfg *config.Config

	dbClient  db.Client
	txManager db.TxManager
	// s3Client  *s3.Client

	MediaHandler *http.ImageHandler
	MediaService *usecase.MediaService
	MediaRepo    *postgres.MediaRepo
}

func LogQuery(ctx context.Context, q db.Query, args ...interface{}) {
	prettyQuery := prettier.Pretty(q.QueryRaw, prettier.PlaceholderDollar, args...)
	slog.InfoContext(ctx, "", "sql", q.Name,
		slog.String("query", prettyQuery),
	)
}

func (s *serviceProvider) DBClient(ctx context.Context) db.Client {
	if s.dbClient == nil {
		pgLog := func(ctx context.Context, q db.Query, args ...interface{}) {}

		pgLog = LogQuery

		cl, err := pg.New(ctx, s.cfg.Postgres.DSN(), pgLog)
		if err != nil {
			log.Fatalf("failed to create db client: %v", err)
		}

		err = cl.DB().Ping(ctx)
		if err != nil {
			log.Fatalf("ping error: %s", err.Error())
		}
		closer.Add(cl.Close)
		s.dbClient = cl
	}

	return s.dbClient
}

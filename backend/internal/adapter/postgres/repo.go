package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
)

const (
	TableName = "media"

	IDCol        = "id"
	UserIDCol    = "user_id"
	URLCol       = "url"
	ThumbURLCol  = "thumb_url"
	MimeTypeCol  = "mime_type"
	SizeBytesCol = "size_bytes"
	ChecksumCol  = "checksum"
	TakenAtCol   = "datetime_original"
	WidthCol     = "width"
	HeightCol    = "height"
	MakeCol      = "camera_make"
	ModelCol     = "camera_model"
	SoftwareCol  = "software"
	OrientCol    = "orientation"
	CreatedAtCol = "created_at"
)

type MediaRepo struct {
	db               db.Client
	uniqUserChecksum string // name of UNIQUE(user_id, checksum) constraint for error mapping

}

func NewMediaRepo(db db.Client) *MediaRepo {
	return &MediaRepo{
		db:               db,
		uniqUserChecksum: "media_user_checksum_uniq", // as defined in migration file
	}
}

func (r *MediaRepo) Create(ctx context.Context, m *domain.Media) error {
	builder := squirrel.Insert(TableName).
		Columns(
			UserIDCol,
			URLCol,
			ThumbURLCol,
			MimeTypeCol,
			SizeBytesCol,
			ChecksumCol,
			CreatedAtCol,
			TakenAtCol,
			WidthCol,
			HeightCol,
			MakeCol,
			ModelCol,
			SoftwareCol,
			OrientCol,
		).
		Values(
			m.UserID,
			m.URL,
			m.ThumbURL,
			m.MimeType,
			m.SizeBytes,
			m.Checksum,
			m.CreatedAt,
			m.Metadata.DateTimeOriginal,
			m.Metadata.Width,
			m.Metadata.Height,
			m.Metadata.CameraMake,
			m.Metadata.CameraModel,
			m.Metadata.Software,
			m.Metadata.Orientation,
		)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build media insert: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.Create",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == r.uniqUserChecksum {
			return &domain.DomainError{
				Code: domain.ErrMediaAlreadyExists.Code,
				Msg:  domain.ErrMediaAlreadyExists.Msg,
			}
		}
		return err
	}

	return nil
}

func (r *MediaRepo) Get(ctx context.Context, userID, id int64) (*domain.Media, error) {
	builder := squirrel.Select(
		IDCol,
		UserIDCol,
		URLCol,
		ThumbURLCol,
		MimeTypeCol,
		SizeBytesCol,
		ChecksumCol,
		CreatedAtCol,
		TakenAtCol,
		OrientCol,
		WidthCol,
		HeightCol,
		MakeCol,
		ModelCol,
		SoftwareCol,
	).
		From(TableName).
		Where(squirrel.Eq{IDCol: id, UserIDCol: userID}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build media select: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.Get",
		QueryRaw: query,
	}

	row := r.db.DB().QueryRowContext(ctx, q, args...)

	var m domain.Media
	var meta domain.Metadata
	err = row.Scan(
		&m.ID,
		&m.UserID,
		&m.URL,
		&m.ThumbURL,
		&m.MimeType,
		&m.SizeBytes,
		&m.Checksum,
		&m.CreatedAt,
		&meta.DateTimeOriginal,
		&meta.Orientation,
		&meta.Width,
		&meta.Height,
		&meta.CameraMake,
		&meta.CameraModel,
		&meta.Software,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan media: %w", err)
	}

	m.Metadata = meta
	return &m, nil
}

// fetches by user + checksum for exact-byte dedup
func (r *MediaRepo) GetByChecksum(ctx context.Context, userID int64, checksum string) (*domain.Media, error) {
	builder := squirrel.Select(
		IDCol,
		UserIDCol,
		URLCol,
		ThumbURLCol,
		MimeTypeCol,
		SizeBytesCol,
		ChecksumCol,
		CreatedAtCol,
		TakenAtCol,
		WidthCol,
		HeightCol,
		MakeCol,
		ModelCol,
		SoftwareCol,
		OrientCol,
	).
		From(TableName).
		Where(squirrel.Eq{UserIDCol: userID, ChecksumCol: checksum}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build media select: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.GetByChecksum",
		QueryRaw: query,
	}

	row := r.db.DB().QueryRowContext(ctx, q, args...)
	var m domain.Media
	var meta domain.Metadata

	err = row.Scan(
		&m.ID,
		&m.UserID,
		&m.URL,
		&m.ThumbURL,
		&m.MimeType,
		&m.SizeBytes,
		&m.Checksum,
		&m.CreatedAt,
		&meta.DateTimeOriginal,
		&meta.Width,
		&meta.Height,
		&meta.CameraMake,
		&meta.CameraModel,
		&meta.Software,
		&meta.Orientation,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan media: %w", err)
	}
	m.Metadata = meta
	return &m, nil
}

// returns paginated results and the total count
func (r *MediaRepo) List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error) {
	return nil, 0, nil
}

func (r *MediaRepo) Delete(ctx context.Context, userID, id int64) error {
	builder := squirrel.Delete(TableName).
		Where(squirrel.Eq{
			IDCol:     id,
			UserIDCol: userID,
		})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build media delete: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.Delete",
		QueryRaw: query,
	}

	tag, err := r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("exec media delete: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("media not found or not owned by user %d", userID)
	}

	return nil
}

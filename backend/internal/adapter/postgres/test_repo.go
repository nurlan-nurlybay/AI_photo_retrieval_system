package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
)

type Repo struct {
	db               db.Client
	uniqUserChecksum string // name of UNIQUE(user_id, checksum) constraint for error mapping

}

func NewMediaRepo(db db.Client) *Repo {
	return &Repo{
		db:               db,
		uniqUserChecksum: "media_user_checksum_uniq", // as defined in migration file
	}
}

const (
	mediaTableName = "media"

	mediaIDCol           = "id"
	mediaUserIDCol       = "user_id"
	mediaURLCol          = "url"
	mediaThumbURLCol     = "thumb_url"
	mediaMimeTypeCol     = "mime_type"
	mediaSizeBytesCol    = "size_bytes"
	mediaChecksumCol     = "checksum"
	mediaCreatedAtCol    = "created_at"
	mediaTakenAtCol      = "taken_at"
	mediaMetaWidthCol    = "meta_width"
	mediaMetaHeightCol   = "meta_height"
	mediaMetaFmtCol      = "meta_file_format"
	mediaMetaMakeCol     = "meta_camera_make"
	mediaMetaModelCol    = "meta_camera_model"
	mediaMetaSoftwareCol = "meta_software"
	mediaMetaOrientCol   = "meta_orientation"
)

func (r *Repo) Create(ctx context.Context, m *domain.Media) error {
	builder := sq.Insert(mediaTableName).
		Columns(
			mediaIDCol, mediaUserIDCol, mediaURLCol, mediaThumbURLCol,
			mediaMimeTypeCol, mediaSizeBytesCol, mediaChecksumCol,
			mediaCreatedAtCol, mediaTakenAtCol,
			mediaMetaWidthCol, mediaMetaHeightCol, mediaMetaFmtCol,
			mediaMetaMakeCol, mediaMetaModelCol, mediaMetaSoftwareCol, mediaMetaOrientCol,
		).
		Values(
			m.ID, m.UserID, m.URL, m.ThumbURL,
			m.MimeType, m.SizeBytes, m.Checksum,
			m.CreatedAt, m.Metadata.DateTimeOriginal,
			m.Metadata.Width, m.Metadata.Height, m.Metadata.FileFormat,
			m.Metadata.CameraMake, m.Metadata.CameraModel, m.Metadata.Software, m.Metadata.Orientation,
		)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build media insert: %w", err)
	}

	q := db.Query{
		Name:     "MediaPG.Create",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		// map unique_constrain err to domain error 
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

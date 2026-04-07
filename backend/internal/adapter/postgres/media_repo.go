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
	LocalPathCol = "local_path"
<<<<<<< HEAD
=======
	StatusCol    = "status"
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
)

var _ domain.MediaRepository = (*MediaRepo)(nil)

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

func (r *MediaRepo) Create(ctx context.Context, m *domain.Media) (int64, error) {
	query, args, err := squirrel.
		Insert(TableName).
		Columns(
			UserIDCol, URLCol, ThumbURLCol, MimeTypeCol, SizeBytesCol,
			ChecksumCol, CreatedAtCol, TakenAtCol, WidthCol, HeightCol,
<<<<<<< HEAD
			MakeCol, ModelCol, SoftwareCol, OrientCol, LocalPathCol,
=======
			MakeCol, ModelCol, SoftwareCol, OrientCol, LocalPathCol, StatusCol,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		).
		Values(
			m.UserID, m.URL, m.ThumbURL, m.MimeType, m.SizeBytes,
			m.Checksum, m.CreatedAt, m.Metadata.DateTimeOriginal,
			m.Metadata.Width, m.Metadata.Height,
			m.Metadata.CameraMake, m.Metadata.CameraModel,
<<<<<<< HEAD
			m.Metadata.Software, m.Metadata.Orientation, m.LocalPath,
=======
			m.Metadata.Software, m.Metadata.Orientation, m.LocalPath, m.Status,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build media insert: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.Create",
		QueryRaw: query,
	}

	var id int64
	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == r.uniqUserChecksum {
			return 0, &domain.DomainError{
				Code: domain.ErrMediaAlreadyExists.Code,
				Msg:  domain.ErrMediaAlreadyExists.Msg,
			}
		}
		return 0, fmt.Errorf("exec media insert: %w", err)
	}

	return id, nil
}

func (r *MediaRepo) Get(ctx context.Context, userID, mediaID int64) (*domain.Media, error) {
	query, args, err := squirrel.
		Select(
			IDCol, UserIDCol, URLCol, ThumbURLCol, MimeTypeCol, SizeBytesCol,
			ChecksumCol, CreatedAtCol, TakenAtCol, OrientCol, WidthCol, HeightCol,
<<<<<<< HEAD
			MakeCol, ModelCol, SoftwareCol, LocalPathCol,
=======
			MakeCol, ModelCol, SoftwareCol, LocalPathCol, StatusCol,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		).
		From(TableName).
		Where(squirrel.Eq{IDCol: mediaID, UserIDCol: userID}).
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
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
		&m.ID, &m.UserID, &m.URL, &m.ThumbURL, &m.MimeType, &m.SizeBytes,
		&m.Checksum, &m.CreatedAt, &meta.DateTimeOriginal, &meta.Orientation,
		&meta.Width, &meta.Height, &meta.CameraMake, &meta.CameraModel, &meta.Software,
<<<<<<< HEAD
		&m.LocalPath,
=======
		&m.LocalPath, &m.Status,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
	query, args, err := squirrel.
		Select(
			IDCol, UserIDCol, URLCol, ThumbURLCol, MimeTypeCol, SizeBytesCol,
			ChecksumCol, CreatedAtCol, TakenAtCol, WidthCol, HeightCol,
<<<<<<< HEAD
			MakeCol, ModelCol, SoftwareCol, OrientCol, LocalPathCol,
=======
			MakeCol, ModelCol, SoftwareCol, OrientCol, LocalPathCol, StatusCol,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		).
		From(TableName).
		Where(squirrel.Eq{UserIDCol: userID, ChecksumCol: checksum}).
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
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
		&m.ID, &m.UserID, &m.URL, &m.ThumbURL, &m.MimeType, &m.SizeBytes,
		&m.Checksum, &m.CreatedAt, &meta.DateTimeOriginal, &meta.Width,
		&meta.Height, &meta.CameraMake, &meta.CameraModel, &meta.Software,
<<<<<<< HEAD
		&meta.Orientation, &m.LocalPath,
=======
		&meta.Orientation, &m.LocalPath, &m.Status,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
	builder := squirrel.
		Select(
			IDCol, UserIDCol, URLCol, ThumbURLCol, MimeTypeCol,
			SizeBytesCol, ChecksumCol, CreatedAtCol, TakenAtCol,
			WidthCol, HeightCol, MakeCol, ModelCol, SoftwareCol,
<<<<<<< HEAD
			OrientCol, LocalPathCol,
=======
			OrientCol, LocalPathCol, StatusCol,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		).
		From(TableName).
		PlaceholderFormat(squirrel.Dollar)

	// --- Filters ---
	if f.UserID != nil {
		builder = builder.Where(squirrel.Eq{UserIDCol: *f.UserID})
	}
	if f.After != nil {
		builder = builder.Where(squirrel.GtOrEq{CreatedAtCol: *f.After})
	}
	if f.Before != nil {
		builder = builder.Where(squirrel.LtOrEq{CreatedAtCol: *f.Before})
	}
	if len(f.MimeLike) > 0 {
		likes := make([]string, 0, len(f.MimeLike))
		for _, v := range f.MimeLike {
			likes = append(likes, "%"+v+"%")
		}
		or := squirrel.Or{}
		for _, pattern := range likes {
			or = append(or, squirrel.Like{MimeTypeCol: pattern})
		}
		builder = builder.Where(or)
	}

	// --- Sorting ---
	if s.Field != "" {
		dir := "ASC"
		if s.Desc {
			dir = "DESC"
		}
		builder = builder.OrderBy(fmt.Sprintf("%s %s", s.Field, dir))
	} else {
		builder = builder.OrderBy(fmt.Sprintf("%s DESC", CreatedAtCol))
	}

	// --- Pagination ---
	if p.Limit > 0 {
		builder = builder.Limit(uint64(p.Limit))
	}
	if p.Offset > 0 {
		builder = builder.Offset(uint64(p.Offset))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build media list: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.List",
		QueryRaw: query,
	}

	rows, err := r.db.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("exec media list: %w", err)
	}
	defer rows.Close()

	var results []*domain.Media
	for rows.Next() {
		var m domain.Media
		var meta domain.Metadata

		if err := rows.Scan(
			&m.ID, &m.UserID, &m.URL, &m.ThumbURL, &m.MimeType,
			&m.SizeBytes, &m.Checksum, &m.CreatedAt,
			&meta.DateTimeOriginal, &meta.Width, &meta.Height,
			&meta.CameraMake, &meta.CameraModel, &meta.Software,
<<<<<<< HEAD
			&meta.Orientation, &m.LocalPath,
=======
			&meta.Orientation, &m.LocalPath, &m.Status,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		); err != nil {
			return nil, 0, fmt.Errorf("scan media: %w", err)
		}

		m.Metadata = meta
		results = append(results, &m)
	}

	// --- Total count ---
	countBuilder := squirrel.
		Select("COUNT(*)").
		From(TableName).
		PlaceholderFormat(squirrel.Dollar)

	if f.UserID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{UserIDCol: *f.UserID})
	}
	if f.After != nil {
		countBuilder = countBuilder.Where(squirrel.GtOrEq{CreatedAtCol: *f.After})
	}
	if f.Before != nil {
		countBuilder = countBuilder.Where(squirrel.LtOrEq{CreatedAtCol: *f.Before})
	}
	if len(f.MimeLike) > 0 {
		or := squirrel.Or{}
		for _, v := range f.MimeLike {
			or = append(or, squirrel.Like{MimeTypeCol: "%" + v + "%"})
		}
		countBuilder = countBuilder.Where(or)
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build media count: %w", err)
	}

	countQ := db.Query{
		Name:     "MediaRepo.ListCount",
		QueryRaw: countQuery,
	}

	var total int
	if err := r.db.DB().QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("exec media count: %w", err)
	}

	return results, total, nil
}

func (r *MediaRepo) Delete(ctx context.Context, userID, id int64) error {
	query, args, err := squirrel.
		Delete(TableName).
		Where(squirrel.Eq{IDCol: id, UserIDCol: userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
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
<<<<<<< HEAD
=======

func (r *MediaRepo) UpdateStatus(ctx context.Context, userID, id int64, status string) error {
	query, args, err := squirrel.
		Update(TableName).
		Set(StatusCol, status).
		Where(squirrel.Eq{IDCol: id, UserIDCol: userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build media update status: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.UpdateStatus",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("exec media update status: %w", err)
	}

	return nil
}

func (r *MediaRepo) ListAllByUser(ctx context.Context, userID int64) ([]*domain.Media, error) {
	query, args, err := squirrel.
		Select(
			IDCol, UserIDCol, URLCol, ThumbURLCol, MimeTypeCol,
			SizeBytesCol, ChecksumCol, CreatedAtCol, TakenAtCol,
			WidthCol, HeightCol, MakeCol, ModelCol, SoftwareCol,
			OrientCol, LocalPathCol, StatusCol,
		).
		From(TableName).
		Where(squirrel.Eq{UserIDCol: userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build media list all by user: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.ListAllByUser",
		QueryRaw: query,
	}

	rows, err := r.db.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("exec media list all by user: %w", err)
	}
	defer rows.Close()

	var results []*domain.Media
	for rows.Next() {
		var m domain.Media
		var meta domain.Metadata

		if err := rows.Scan(
			&m.ID, &m.UserID, &m.URL, &m.ThumbURL, &m.MimeType,
			&m.SizeBytes, &m.Checksum, &m.CreatedAt,
			&meta.DateTimeOriginal, &meta.Width, &meta.Height,
			&meta.CameraMake, &meta.CameraModel, &meta.Software,
			&meta.Orientation, &m.LocalPath, &m.Status,
		); err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}

		m.Metadata = meta
		results = append(results, &m)
	}

	return results, nil
}

func (r *MediaRepo) DeleteAllByUser(ctx context.Context, userID int64) error {
	query, args, err := squirrel.
		Delete(TableName).
		Where(squirrel.Eq{UserIDCol: userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build media delete all by user: %w", err)
	}

	q := db.Query{
		Name:     "MediaRepo.DeleteAllByUser",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("exec media delete all by user: %w", err)
	}

	return nil
}
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1

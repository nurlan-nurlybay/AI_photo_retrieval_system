package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
)

type MediaService interface {
	UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error)
	Delete(ctx context.Context, userID, id int64) error
	GetByID(ctx context.Context, userID, id int64) (*domain.Media, error)
	List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error)
}

type (
	ObjectStorage interface {
		Put(ctx context.Context, key string, r *bytes.Reader) (publicURL string, err error)
		Delete(ctx context.Context, key string) error
	}

	ImageProcessor interface {
		Process(original []byte) (oriented []byte, thumb []byte, width int, height int, err error)
	}

	MetadataExtractor interface {
		Extract(buf []byte) (ExtractedMetadata, error)
	}

	ExtractedMetadata struct {
		DateTimeOriginal *time.Time
		Orientation      int
		CameraMake       string
		CameraModel      string
		Software         string
	}
)

type mediaService struct {
	repo  domain.MediaRepository
	store ObjectStorage
	img   ImageProcessor
	meta  MetadataExtractor
	clock func() time.Time
}

func NewMediaService(
	repo domain.MediaRepository,
	store ObjectStorage,
	img ImageProcessor,
	meta MetadataExtractor,
) MediaService {
	return &mediaService{
		repo:  repo,
		store: store,
		img:   img,
		meta:  meta,
		clock: time.Now,
	}
}

func (s *mediaService) UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error) {
	out := make([]ucdto.UploadResult, 0, len(items))

	for _, it := range items {
		// Read all inmemory TODO: streaming/tempfile later
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(it.Body); err != nil {
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Exact-byte checksum for idempotency
		sum := sha256.Sum256(buf.Bytes())
		check := hex.EncodeToString(sum[:])

		// Dedup per user (best-effort)
		if existing, _ := s.repo.GetByChecksum(ctx, it.UserID, check); existing != nil {
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusDuplicate, Media: existing})
			continue
		}

		// Extract metadata (best-effort)
		meta, _ := s.meta.Extract(buf.Bytes())

		// Auto-orient and make thumb.
		oriented, thumb, w, h, err := s.img.Process(buf.Bytes())
		if err != nil {
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Store objects. Keys can be anything deterministic
		keyOrig := fmt.Sprintf("media/%d/%s/original", it.UserID, check)
		keyThumb := fmt.Sprintf("media/%d/%s/thumb.jpg", it.UserID, check)

		origURL, err := s.store.Put(ctx, keyOrig, bytes.NewReader(oriented))
		if err != nil {
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		thumbURL, err := s.store.Put(ctx, keyThumb, bytes.NewReader(thumb))
		if err != nil {
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Build domain model and persist.
		m := &domain.Media{
			UserID:    it.UserID,
			URL:       origURL,
			ThumbURL:  thumbURL,
			MimeType:  it.MimeType,
			SizeBytes: int64(len(oriented)),
			Checksum:  check,
			CreatedAt: s.clock().UTC(),
			Metadata: domain.Metadata{
				DateTimeOriginal: meta.DateTimeOriginal,
				Orientation:      meta.Orientation,
				Width:            w,
				Height:           h,
				FileFormat:       extFromMime(it.MimeType),
				CameraMake:       meta.CameraMake,
				CameraModel:      meta.CameraModel,
				Software:         meta.Software,
			},
		}

		if err := s.repo.Create(ctx, m); err != nil {
			_ = s.store.Delete(ctx, m.URL)
			_ = s.store.Delete(ctx, m.ThumbURL)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		out = append(out, ucdto.UploadResult{Status: ucdto.StatusSaved, Media: m})
	}

	return out, nil
}

func (s *mediaService) Delete(ctx context.Context, userID, id int64) error {
	// fetch first to know storage keys
	_, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	// TODO: delete from object storage if you store keys somewhere accessible.
	// _ = s.store.Delete(ctx, keyOrig)
	// _ = s.store.Delete(ctx, keyThumb)

	return s.repo.Delete(ctx, userID, id)
}

func (s *mediaService) GetByID(ctx context.Context, userID, id int64) (*domain.Media, error) {
	m, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *mediaService) List(
	ctx context.Context,
	f domain.MediaFilter,
	p domain.Page,
	so domain.Sort,
) ([]*domain.Media, int, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	items, total, err := s.repo.List(ctx, f, p, so)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func extFromMime(mt string) string {
	switch mt {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/heic", "image/heif":
		return "heic"
	default:
		return "unsupported media type"
	}
}

var _ MediaService = (*mediaService)(nil)

package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type (
	MediaService interface {
		UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error)
		Delete(ctx context.Context, userID, id int64) error
		GetByID(ctx context.Context, userID, id int64) (*domain.Media, error)
		List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error)
	}

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

	Queue interface {
		Enqueue(ctx context.Context, key string, payload []byte) error
	}
)

type mediaService struct {
	repo  domain.MediaRepository
	store ObjectStorage
	queue Queue // wraps Redis
	img   ImageProcessor
	meta  MetadataExtractor
	clock func() time.Time
	log   *logger.Logger
}

func NewMediaService(
	repo domain.MediaRepository,
	store ObjectStorage,
	queue Queue,
	img ImageProcessor,
	meta MetadataExtractor,
	log *logger.Logger,
) MediaService {
	return &mediaService{
		repo:  repo,
		store: store,
		queue: queue,
		img:   img,
		meta:  meta,
		clock: time.Now,
		log:   log,
	}
}

func (s *mediaService) UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error) {
	s.log.InfoContext(ctx, "starting upload batch", "count", len(items))
	out := make([]ucdto.UploadResult, 0, len(items))

	for i, it := range items {
		s.log.DebugContext(ctx, "processing upload item",
			"index", i,
			"filename", it.Filename,
			"user_id", it.UserID,
			"mime_type", it.MimeType,
		)

		// Read all inmemory
		// TODO: streaming/tempfile later
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(it.Body); err != nil {
			s.log.ErrorContext(ctx, "failed to read request body", "index", i, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		s.log.DebugContext(ctx, "read upload body", "index", i, "bytes", buf.Len())

		// Compute checksum for idempotency
		sum := sha256.Sum256(buf.Bytes())
		check := hex.EncodeToString(sum[:])
		s.log.DebugContext(ctx, "checksum computed", "index", i, "checksum", check)

		// Dedup check
		existing, err := s.repo.GetByChecksum(ctx, it.UserID, check)
		if err != nil {
			s.log.ErrorContext(ctx, "dedup lookup failed", "index", i, "error", err)
		}
		if existing != nil {
			s.log.DebugContext(ctx, "duplicate media found, skipping", "index", i, "media_id", existing.ID)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusDuplicate, Media: existing})
			continue
		}

		// Metadata extraction
		meta, _ := s.meta.Extract(buf.Bytes())
		s.log.DebugContext(ctx, "metadata extracted",
			"index", i,
			"orientation", meta.Orientation,
			"camera_make", meta.CameraMake,
			"camera_model", meta.CameraModel,
		)

		// Image processing (auto orient and make thumb)
		oriented, thumb, w, h, err := s.img.Process(buf.Bytes())
		if err != nil {
			s.log.ErrorContext(ctx, "image processing failed", "index", i, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		s.log.DebugContext(ctx, "image processed", "index", i, "width", w, "height", h)

		// Store original img
		keyOrig := fmt.Sprintf("media/%d/%s/original.%s", it.UserID, check, utils.ExtFromMime(it.MimeType))
		origURL, err := s.store.Put(ctx, keyOrig, bytes.NewReader(oriented))
		if err != nil {
			s.log.ErrorContext(ctx, "failed to store original", "index", i, "key", keyOrig, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		s.log.DebugContext(ctx, "original stored", "index", i, "url", origURL)

		// Store thumb
		keyThumb := fmt.Sprintf("media/%d/%s/thumb.jpg", it.UserID, check) // hardcoded by vips bimg.JPEG
		thumbURL, err := s.store.Put(ctx, keyThumb, bytes.NewReader(thumb))
		if err != nil {
			s.log.ErrorContext(ctx, "failed to store thumbnail", "index", i, "key", keyThumb, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		s.log.DebugContext(ctx, "thumbnail stored", "index", i, "url", thumbURL)

		// Build domain model and persist in DB
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
				CameraMake:       meta.CameraMake,
				CameraModel:      meta.CameraModel,
				Software:         meta.Software,
			},
		}

		if err := s.repo.Create(ctx, m); err != nil {
			s.log.ErrorContext(ctx, "failed to persist media", "index", i, "error", err)
			_ = s.store.Delete(ctx, m.URL)
			_ = s.store.Delete(ctx, m.ThumbURL)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		//  enqueue embedding job
		job := ucdto.EmbedJob{UserID: m.UserID, MediaID: m.ID, Modality: "image"}
		b, _ := json.Marshal(job)
		if err := s.queue.Enqueue(ctx, "jobs:embed", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "job enqueued", "media_id", m.ID, "queue", "jobs:embed")
		}

		out = append(out, ucdto.UploadResult{Status: ucdto.StatusSaved, Media: m})
	}

	s.log.InfoContext(ctx, "upload batch completed", "total", len(items), "results", len(out))

	return out, nil
}

func (s *mediaService) Delete(ctx context.Context, userID, id int64) error {
	_, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	// TODO: delete from object storage
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

var _ MediaService = (*mediaService)(nil)

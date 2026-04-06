package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type MediaService interface {
	UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error)
	Delete(ctx context.Context, userID, mediaID int64) error
	DeleteBatch(ctx context.Context, userID int64, imageIDs []int64) error
	ClearGallery(ctx context.Context, userID int64) error
	DeleteAccount(ctx context.Context, userID int64) error
	GetByID(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
	List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error)
}

type (
	ObjectStorage interface {
		Put(ctx context.Context, key string, r *bytes.Reader) (publicURL string, err error)
		Delete(ctx context.Context, key string) error
		GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
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
	vectorClient VectorClient
	repo         domain.MediaRepository
	store        ObjectStorage
	queue        Queue // wraps Redis
	img          ImageProcessor
	meta         MetadataExtractor
	clock        func() time.Time
	log          *logger.Logger
}

func NewMediaService(
	vc VectorClient,
	repo domain.MediaRepository,
	store ObjectStorage,
	queue Queue,
	img ImageProcessor,
	meta MetadataExtractor,
	log *logger.Logger,
) MediaService {
	return &mediaService{
		vectorClient: vc,
		repo:         repo,
		store:        store,
		queue:        queue,
		img:          img,
		meta:         meta,
		clock:        time.Now,
		log:          log,
	}
}

func (s *mediaService) UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error) {
	s.log.InfoContext(ctx, "starting upload batch", "count", len(items))
	out := make([]ucdto.UploadResult, 0, len(items))

	for i, it := range items {
		// Read all inmemory
		// TODO: streaming/tempfile
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(it.Body); err != nil {
			s.log.ErrorContext(ctx, "failed to read request body", "index", i, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Compute checksum for idempotency
		sum := sha256.Sum256(buf.Bytes())
		checksum := hex.EncodeToString(sum[:])

		// Dedup check
		existing, err := s.repo.GetByChecksum(ctx, it.UserID, checksum)
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

		// Store original img
		keyOrig := fmt.Sprintf("media/%d/%s/original.%s", it.UserID, checksum, utils.ExtFromMime(it.MimeType))
		origURL, err := s.store.Put(ctx, keyOrig, bytes.NewReader(oriented))
		if err != nil {
			s.log.ErrorContext(ctx, "failed to store original", "index", i, "key", keyOrig, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Store thumb
		keyThumb := fmt.Sprintf("media/%d/%s/thumb.jpg", it.UserID, checksum) // hardcoded by vips bimg.JPEG
		thumbURL, err := s.store.Put(ctx, keyThumb, bytes.NewReader(thumb))
		if err != nil {
			s.log.ErrorContext(ctx, "failed to store thumbnail", "index", i, "key", keyThumb, "error", err)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		// Build domain model and persist in DB
		m := &domain.Media{
			UserID:    it.UserID,
			URL:       origURL,
			ThumbURL:  thumbURL,
			MimeType:  it.MimeType,
			SizeBytes: int64(len(oriented)),
			Checksum:  checksum,
			CreatedAt: s.clock().UTC(),
			LocalPath: it.LocalPath,
			Status:    "active",
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

		mediaID, err := s.repo.Create(ctx, m)
		if err != nil {
			s.log.ErrorContext(ctx, "failed to persist media", "index", i, "error", err)
			_ = s.store.Delete(ctx, keyOrig)
			_ = s.store.Delete(ctx, keyThumb)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		m.ID = mediaID

		job := ucdto.EmbedJob{
			UserID:   m.UserID,
			MediaID:  mediaID,
			Modality: "image",
		}
		b, _ := json.Marshal(job)
		if err := s.queue.Enqueue(ctx, "jobs:fast_queue", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue fast embed job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "fast embed job enqueued", "media_id", m.ID)
		}

		if err := s.queue.Enqueue(ctx, "jobs:slow_queue", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue slow embed job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "slow embed job enqueued", "media_id", m.ID)
		}

		out = append(out, ucdto.UploadResult{Status: ucdto.StatusSaved, Media: m})
	}

	s.log.InfoContext(ctx, "upload batch completed", "count", len(items), "saved", len(out))

	return out, nil
}

// Delete removes a single image. Cascade: S3 -> Postgres -> Vector Service.
func (s *mediaService) Delete(ctx context.Context, userID, mediaID int64) error {
	media, err := s.repo.Get(ctx, userID, mediaID)
	if err != nil {
		return err
	}
	if media == nil {
		return fmt.Errorf("media %d not found for user %d", mediaID, userID)
	}

	// 1. S3
	_ = s.store.Delete(ctx, extractKey(media.URL))
	_ = s.store.Delete(ctx, extractKey(media.ThumbURL))

	// 2. Postgres (embeddings cascade via FK)
	if err := s.repo.Delete(ctx, userID, mediaID); err != nil {
		return err
	}

	// 3. Vector Service
	_ = s.vectorClient.DeleteItems(ctx, userID, []int64{mediaID})

	return nil
}

// DeleteBatch removes specific image IDs. Cascade: S3 -> Postgres -> Vector Service.
func (s *mediaService) DeleteBatch(ctx context.Context, userID int64, imageIDs []int64) error {
	s.log.InfoContext(ctx, "delete batch started", "user_id", userID, "count", len(imageIDs))

	// 1. Fetch S3 keys for each image and delete from S3
	for _, id := range imageIDs {
		media, err := s.repo.Get(ctx, userID, id)
		if err != nil {
			s.log.ErrorContext(ctx, "delete batch: failed to fetch media", "media_id", id, "error", err)
			continue
		}
		if media == nil {
			continue
		}
		_ = s.store.Delete(ctx, extractKey(media.URL))
		_ = s.store.Delete(ctx, extractKey(media.ThumbURL))
	}

	// 2. Postgres — delete one by one to respect ownership
	for _, id := range imageIDs {
		if err := s.repo.Delete(ctx, userID, id); err != nil {
			s.log.ErrorContext(ctx, "delete batch: postgres delete failed", "media_id", id, "error", err)
		}
	}

	// 3. Vector Service — single batch call
	if err := s.vectorClient.DeleteItems(ctx, userID, imageIDs); err != nil {
		s.log.ErrorContext(ctx, "delete batch: vector delete failed", "error", err)
	}

	s.log.InfoContext(ctx, "delete batch completed", "user_id", userID, "count", len(imageIDs))
	return nil
}

// ClearGallery removes all user images but keeps the account. Cascade: S3 -> Postgres -> Vector Service.
func (s *mediaService) ClearGallery(ctx context.Context, userID int64) error {
	s.log.InfoContext(ctx, "clear gallery started", "user_id", userID)

	// 1. Fetch all media to get S3 keys
	allMedia, err := s.repo.ListAllByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("clear gallery: list media: %w", err)
	}

	// 2. Delete from S3
	for _, m := range allMedia {
		_ = s.store.Delete(ctx, extractKey(m.URL))
		_ = s.store.Delete(ctx, extractKey(m.ThumbURL))
	}

	// 3. Postgres — bulk delete (embeddings cascade via FK)
	if err := s.repo.DeleteAllByUser(ctx, userID); err != nil {
		return fmt.Errorf("clear gallery: postgres delete: %w", err)
	}

	// 4. Vector Service — clear namespace (drops and recreates collections)
	if err := s.vectorClient.ClearNamespace(ctx, userID); err != nil {
		s.log.ErrorContext(ctx, "clear gallery: vector clear failed", "error", err)
	}

	s.log.InfoContext(ctx, "clear gallery completed", "user_id", userID, "deleted_count", len(allMedia))
	return nil
}

// DeleteAccount removes all data AND nukes the vector namespace. Cascade: S3 -> Postgres -> Vector Service.
func (s *mediaService) DeleteAccount(ctx context.Context, userID int64) error {
	s.log.InfoContext(ctx, "delete account started", "user_id", userID)

	// 1. Fetch all media to get S3 keys
	allMedia, err := s.repo.ListAllByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete account: list media: %w", err)
	}

	// 2. Delete from S3
	for _, m := range allMedia {
		_ = s.store.Delete(ctx, extractKey(m.URL))
		_ = s.store.Delete(ctx, extractKey(m.ThumbURL))
	}

	// 3. Postgres — bulk delete (embeddings cascade via FK)
	if err := s.repo.DeleteAllByUser(ctx, userID); err != nil {
		return fmt.Errorf("delete account: postgres delete: %w", err)
	}

	// 4. Vector Service — nuke namespace (permanent drop)
	if err := s.vectorClient.NukeNamespace(ctx, userID); err != nil {
		s.log.ErrorContext(ctx, "delete account: vector nuke failed", "error", err)
	}

	s.log.InfoContext(ctx, "delete account completed", "user_id", userID, "deleted_count", len(allMedia))
	return nil
}

func (s *mediaService) GetByID(ctx context.Context, userID, mediaID int64) (*domain.Media, error) {
	m, err := s.repo.Get(ctx, userID, mediaID)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// TODO: add sort and filter by time/mimetype
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

func extractKey(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(u.Path, "/")
}

var _ MediaService = (*mediaService)(nil)

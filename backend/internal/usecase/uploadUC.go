package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
<<<<<<< HEAD
=======
	"net/url"
	"strings"
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type MediaService interface {
	UploadBatch(ctx context.Context, items []ucdto.UploadInput) ([]ucdto.UploadResult, error)
	Delete(ctx context.Context, userID, mediaID int64) error
<<<<<<< HEAD
=======
	DeleteBatch(ctx context.Context, userID int64, imageIDs []int64) error
	ClearGallery(ctx context.Context, userID int64) error
	DeleteAccount(ctx context.Context, userID int64) error
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	GetByID(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
	List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error)
}

type (
	ObjectStorage interface {
		Put(ctx context.Context, key string, r *bytes.Reader) (publicURL string, err error)
		Delete(ctx context.Context, key string) error
<<<<<<< HEAD
=======
		GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
<<<<<<< HEAD
	vectorIndex VectorIndex

	repo  domain.MediaRepository
	store ObjectStorage
	queue Queue // wraps Redis
	img   ImageProcessor
	meta  MetadataExtractor
	clock func() time.Time
	log   *logger.Logger
}

func NewMediaService(
	milvus VectorIndex,

=======
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	repo domain.MediaRepository,
	store ObjectStorage,
	queue Queue,
	img ImageProcessor,
	meta MetadataExtractor,
	log *logger.Logger,
) MediaService {
	return &mediaService{
<<<<<<< HEAD
		vectorIndex: milvus,

		repo:  repo,
		store: store,
		queue: queue,
		img:   img,
		meta:  meta,
		clock: time.Now,
		log:   log,
=======
		vectorClient: vc,
		repo:         repo,
		store:        store,
		queue:        queue,
		img:          img,
		meta:         meta,
		clock:        time.Now,
		log:          log,
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
<<<<<<< HEAD
=======
			Status:    "active",
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
<<<<<<< HEAD
			_ = s.store.Delete(ctx, m.URL)
			_ = s.store.Delete(ctx, m.ThumbURL)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}

		//  enqueue embedding job
=======
			_ = s.store.Delete(ctx, keyOrig)
			_ = s.store.Delete(ctx, keyThumb)
			out = append(out, ucdto.UploadResult{Status: ucdto.StatusFailed, Err: err})
			continue
		}
		m.ID = mediaID

>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		job := ucdto.EmbedJob{
			UserID:   m.UserID,
			MediaID:  mediaID,
			Modality: "image",
		}
		b, _ := json.Marshal(job)
<<<<<<< HEAD
		if err := s.queue.Enqueue(ctx, "jobs:embed", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "job enqueued", "media_id", m.ID, "queue", "jobs:embed")
=======
		if err := s.queue.Enqueue(ctx, "jobs:fast_queue", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue fast embed job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "fast embed job enqueued", "media_id", m.ID)
		}

		if err := s.queue.Enqueue(ctx, "jobs:slow_queue", b); err != nil {
			s.log.ErrorContext(ctx, "failed to enqueue slow embed job", "media_id", m.ID, "error", err)
		} else {
			s.log.DebugContext(ctx, "slow embed job enqueued", "media_id", m.ID)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		}

		out = append(out, ucdto.UploadResult{Status: ucdto.StatusSaved, Media: m})
	}

	s.log.InfoContext(ctx, "upload batch completed", "count", len(items), "saved", len(out))

	return out, nil
}

<<<<<<< HEAD
=======
// Delete removes a single image. Cascade: S3 -> Postgres -> Vector Service.
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
func (s *mediaService) Delete(ctx context.Context, userID, mediaID int64) error {
	media, err := s.repo.Get(ctx, userID, mediaID)
	if err != nil {
		return err
	}
<<<<<<< HEAD

	// TODO: handle delete from milvus, pg, seaweedfs
	_ = s.vectorIndex.Delete(ctx, mediaID)

	_ = s.store.Delete(ctx, media.URL)
	_ = s.store.Delete(ctx, media.ThumbURL)

	_ = s.repo.Delete(ctx, userID, mediaID)
=======
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1

	return nil
}

<<<<<<< HEAD
=======
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

>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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

<<<<<<< HEAD
=======
func extractKey(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(u.Path, "/")
}

>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
var _ MediaService = (*mediaService)(nil)

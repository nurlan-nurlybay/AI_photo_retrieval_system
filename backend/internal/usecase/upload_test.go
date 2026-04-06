package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mocks ────────────────────────────────────────────────────────────

type mockStorage struct {
	putFn    func(ctx context.Context, key string, r *bytes.Reader) (string, error)
	deleteFn func(ctx context.Context, key string) error
	putCalls []putCall
	delCalls []string
}

type putCall struct {
	key  string
	data []byte
}

func (m *mockStorage) Put(ctx context.Context, key string, r *bytes.Reader) (string, error) {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r)
	m.putCalls = append(m.putCalls, putCall{key: key, data: buf.Bytes()})
	if m.putFn != nil {
		return m.putFn(ctx, key, bytes.NewReader(buf.Bytes()))
	}
	return fmt.Sprintf("http://test-bucket.s3.amazonaws.com/%s", key), nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.delCalls = append(m.delCalls, key)
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockStorage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return fmt.Sprintf("http://test-bucket.s3.amazonaws.com/%s?expires=%v", key, expiration), nil
}

type mockRepo struct {
	getByChecksumFn func(ctx context.Context, userID int64, checksum string) (*domain.Media, error)
	createFn        func(ctx context.Context, m *domain.Media) (int64, error)
	getFn           func(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
	deleteFn        func(ctx context.Context, userID, id int64) error
	listFn          func(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error)
	nextID          int64
}

func (m *mockRepo) GetByChecksum(ctx context.Context, userID int64, checksum string) (*domain.Media, error) {
	if m.getByChecksumFn != nil {
		return m.getByChecksumFn(ctx, userID, checksum)
	}
	return nil, nil
}

func (m *mockRepo) Create(ctx context.Context, media *domain.Media) (int64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, media)
	}
	m.nextID++
	return m.nextID, nil
}

func (m *mockRepo) Get(ctx context.Context, userID, mediaID int64) (*domain.Media, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID, mediaID)
	}
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, userID, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, id)
	}
	return nil
}

func (m *mockRepo) List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f, p, s)
	}
	return nil, 0, nil
}

type mockQueue struct {
	enqueued [][]byte
}

func (m *mockQueue) Enqueue(_ context.Context, _ string, payload []byte) error {
	m.enqueued = append(m.enqueued, payload)
	return nil
}

type mockImageProc struct{}

func (m *mockImageProc) Process(original []byte) ([]byte, []byte, int, int, error) {
	return original, []byte("thumb-data"), 1920, 1080, nil
}

type mockMetaExtractor struct{}

func (m *mockMetaExtractor) Extract(_ []byte) (ExtractedMetadata, error) {
	t := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return ExtractedMetadata{
		DateTimeOriginal: &t,
		Orientation:      1,
		CameraMake:       "Apple",
		CameraModel:      "iPhone 15 Pro",
	}, nil
}

type mockVectorClient struct{}

func (m *mockVectorClient) SearchHybrid(_ context.Context, _ string, _ string, _ []float32, _ []float32, _ int) ([]SearchResult, bool, error) {
	return nil, false, nil
}
func (m *mockVectorClient) DeleteImage(_ context.Context, _ string, _ int64) error {
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────

func newTestService(store *mockStorage, repo *mockRepo) (*mediaService, *mockStorage, *mockRepo, *mockQueue) {
	if store == nil {
		store = &mockStorage{}
	}
	if repo == nil {
		repo = &mockRepo{}
	}
	q := &mockQueue{}
	log := logger.New(logger.Config{Level: "debug", Format: "text", SourceFolder: "test"})
	svc := &mediaService{
		vectorClient: &mockVectorClient{},
		repo:        repo,
		store:       store,
		queue:       q,
		img:         &mockImageProc{},
		meta:        &mockMetaExtractor{},
		clock:       func() time.Time { return time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC) },
		log:         log,
	}
	return svc, store, repo, q
}

func makeInput(filename, mime string, data []byte) ucdto.UploadInput {
	return ucdto.UploadInput{
		UserID:   404,
		Filename: filename,
		MimeType: mime,
		Size:     int64(len(data)),
		Body:     bytes.NewReader(data),
	}
}

// ── Tests ────────────────────────────────────────────────────────────

func TestUploadBatch_SingleImage_Success(t *testing.T) {
	svc, store, _, q := newTestService(nil, nil)

	input := makeInput("photo.jpg", "image/jpeg", []byte("raw-image-data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, ucdto.StatusSaved, results[0].Status)
	assert.NotNil(t, results[0].Media)

	// Verify two Put calls: original + thumbnail
	require.Len(t, store.putCalls, 2)
	assert.True(t, strings.HasSuffix(store.putCalls[0].key, "/original.jpg"), "first put should be original")
	assert.True(t, strings.HasSuffix(store.putCalls[1].key, "/thumb.jpg"), "second put should be thumb")

	// Verify embed jobs were enqueued (fast + slow)
	assert.Len(t, q.enqueued, 2)
}

func TestUploadBatch_MediaID_SetAfterCreate(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ *domain.Media) (int64, error) {
			return 42, nil
		},
	}
	svc, _, _, _ := newTestService(nil, repo)

	input := makeInput("photo.jpg", "image/jpeg", []byte("data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)

	media := results[0].Media
	require.NotNil(t, media)
	assert.Equal(t, int64(42), media.ID,
		"Media.ID should be set to the value returned by repo.Create")
}

func TestUploadBatch_Dedup(t *testing.T) {
	existing := &domain.Media{
		ID:       99,
		UserID:   404,
		URL:      "http://test-bucket.s3.amazonaws.com/media/404/existing/original.jpg",
		Checksum: "existing-checksum",
	}
	repo := &mockRepo{
		getByChecksumFn: func(_ context.Context, _ int64, _ string) (*domain.Media, error) {
			return existing, nil
		},
	}
	svc, store, _, q := newTestService(nil, repo)

	input := makeInput("photo.jpg", "image/jpeg", []byte("data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, ucdto.StatusDuplicate, results[0].Status)
	assert.Equal(t, existing, results[0].Media)
	assert.Empty(t, store.putCalls, "no storage calls for duplicates")
	assert.Empty(t, q.enqueued, "no embed jobs for duplicates")
}

func TestUploadBatch_StorageFailure_RollbackNotAttempted(t *testing.T) {
	store := &mockStorage{
		putFn: func(_ context.Context, key string, _ *bytes.Reader) (string, error) {
			if strings.Contains(key, "original") {
				return "", errors.New("storage unavailable")
			}
			return "http://test-bucket.s3.amazonaws.com/" + key, nil
		},
	}
	svc, _, _, q := newTestService(store, nil)

	input := makeInput("photo.jpg", "image/jpeg", []byte("data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err) // batch itself doesn't return error
	require.Len(t, results, 1)
	assert.Equal(t, ucdto.StatusFailed, results[0].Status)
	assert.Contains(t, results[0].Err.Error(), "storage unavailable")
	assert.Empty(t, q.enqueued, "no embed jobs on storage failure")
}

func TestUploadBatch_DBFailure_CleanupStorage(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ *domain.Media) (int64, error) {
			return 0, errors.New("unique violation")
		},
	}
	svc, store, _, _ := newTestService(nil, repo)

	input := makeInput("photo.jpg", "image/jpeg", []byte("data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, ucdto.StatusFailed, results[0].Status)

	require.Len(t, store.delCalls, 2, "should attempt to clean up both original and thumb")
	for _, call := range store.delCalls {
		assert.False(t, strings.HasPrefix(call, "http://"),
			"Delete should use storage key, not full URL; got '%s'", call)
		assert.True(t, strings.HasPrefix(call, "media/"),
			"Delete key should start with 'media/' prefix; got '%s'", call)
	}
}

func TestDelete_ExtractsKeyFromURL(t *testing.T) {
	media := &domain.Media{
		ID:       1,
		UserID:   404,
		URL:      "http://test-bucket.s3.amazonaws.com/media/404/checksum/original.jpg",
		ThumbURL: "http://test-bucket.s3.amazonaws.com/media/404/checksum/thumb.jpg",
	}
	repo := &mockRepo{
		getFn: func(_ context.Context, _, _ int64) (*domain.Media, error) {
			return media, nil
		},
	}
	svc, store, _, _ := newTestService(nil, repo)

	err := svc.Delete(context.Background(), 404, 1)
	require.NoError(t, err)

	require.Len(t, store.delCalls, 2)
	assert.Equal(t, "media/404/checksum/original.jpg", store.delCalls[0],
		"Delete should extract key from URL")
	assert.Equal(t, "media/404/checksum/thumb.jpg", store.delCalls[1],
		"Delete should extract key from URL")
}

func TestUploadBatch_MultipleImages(t *testing.T) {
	svc, store, _, q := newTestService(nil, nil)

	inputs := []ucdto.UploadInput{
		makeInput("a.jpg", "image/jpeg", []byte("aaa")),
		makeInput("b.png", "image/png", []byte("bbb")),
		makeInput("c.jpg", "image/jpeg", []byte("ccc")),
	}
	results, err := svc.UploadBatch(context.Background(), inputs)

	require.NoError(t, err)
	require.Len(t, results, 3)

	for _, r := range results {
		assert.Equal(t, ucdto.StatusSaved, r.Status)
	}
	assert.Len(t, store.putCalls, 6, "2 puts per image (original + thumb)")
	assert.Len(t, q.enqueued, 6, "two embed jobs per image (fast + slow)")
}

func TestUploadBatch_KeyFormat(t *testing.T) {
	svc, store, _, _ := newTestService(nil, nil)

	input := makeInput("photo.jpg", "image/jpeg", []byte("test-data"))
	_, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})
	require.NoError(t, err)

	origKey := store.putCalls[0].key
	thumbKey := store.putCalls[1].key

	assert.True(t, strings.HasPrefix(origKey, "media/404/"), "original key should start with media/{userID}/")
	assert.True(t, strings.HasSuffix(origKey, "/original.jpg"), "original key should end with /original.{ext}")
	assert.True(t, strings.HasPrefix(thumbKey, "media/404/"), "thumb key should start with media/{userID}/")
	assert.True(t, strings.HasSuffix(thumbKey, "/thumb.jpg"), "thumb key should end with /thumb.jpg")
}

func TestUploadBatch_MetadataPopulated(t *testing.T) {
	svc, _, _, _ := newTestService(nil, nil)

	input := makeInput("photo.jpg", "image/jpeg", []byte("test-data"))
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)
	media := results[0].Media
	require.NotNil(t, media)

	assert.Equal(t, 1920, media.Metadata.Width)
	assert.Equal(t, 1080, media.Metadata.Height)
	assert.Equal(t, "Apple", media.Metadata.CameraMake)
	assert.Equal(t, "iPhone 15 Pro", media.Metadata.CameraModel)
	assert.NotNil(t, media.Metadata.DateTimeOriginal)
}

func TestUploadBatch_EmptyBody(t *testing.T) {
	svc, _, _, _ := newTestService(nil, nil)

	input := makeInput("empty.jpg", "image/jpeg", []byte{})
	results, err := svc.UploadBatch(context.Background(), []ucdto.UploadInput{input})

	require.NoError(t, err)
	require.Len(t, results, 1)
	// Empty body still gets processed (checksum of empty bytes is valid)
	assert.Equal(t, ucdto.StatusSaved, results[0].Status)
}

package seaweedfs

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Seaweedfs) {
	t.Helper()
	ts := httptest.NewServer(handler)
	sw := &Seaweedfs{
		baseURL:   ts.URL,
		publicURL: "http://public.example.com",
		client:    ts.Client(),
	}
	return ts, sw
}

func TestPut_Success(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte

	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	})
	defer ts.Close()

	data := []byte("fake-image-data")
	url, err := sw.Put(context.Background(), "media/404/abc123/original.jpg", bytes.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/media/404/abc123/original.jpg", gotPath)
	assert.Equal(t, data, gotBody)
	assert.Equal(t, "http://public.example.com/media/404/abc123/original.jpg", url)
}

func TestPut_DefaultsPublicURLToBaseURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	sw := &Seaweedfs{baseURL: ts.URL, publicURL: ts.URL, client: ts.Client()}

	url, err := sw.Put(context.Background(), "media/1/key/original.jpg", bytes.NewReader([]byte("data")))
	require.NoError(t, err)
	assert.Equal(t, ts.URL+"/media/1/key/original.jpg", url)
}

func TestPut_ServerError(t *testing.T) {
	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("disk full"))
	})
	defer ts.Close()

	_, err := sw.Put(context.Background(), "key", bytes.NewReader([]byte("data")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload failed")
	assert.Contains(t, err.Error(), "disk full")
}

func TestGet_Success(t *testing.T) {
	expected := []byte("image-bytes-here")
	var gotPath string

	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(expected)
	})
	defer ts.Close()

	data, err := sw.Get(context.Background(), "media/404/abc123/original.jpg")
	require.NoError(t, err)
	assert.Equal(t, "/media/404/abc123/original.jpg", gotPath)
	assert.Equal(t, expected, data)
}

func TestGet_NotFound(t *testing.T) {
	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	defer ts.Close()

	_, err := sw.Get(context.Background(), "missing/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get failed")
}

func TestDelete_Success(t *testing.T) {
	var gotMethod, gotPath string

	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	})
	defer ts.Close()

	err := sw.Delete(context.Background(), "media/404/abc123/original.jpg")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/media/404/abc123/original.jpg", gotPath)
}

func TestDelete_ReceivesFullURL_DoublePrefix(t *testing.T) {
	var gotPath string

	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	})
	defer ts.Close()

	// This simulates what uploadUC.Delete currently does — passes media.URL
	// (a full public URL) instead of just the key. The result is a malformed
	// double-prefixed path.
	fullURL := "http://public.example.com/media/404/abc123/original.jpg"
	_ = sw.Delete(context.Background(), fullURL)

	// The path sent to the server should NOT contain the scheme/host prefix.
	// This test documents the bug: the path will be wrong.
	assert.NotEqual(t, "/media/404/abc123/original.jpg", gotPath,
		"BUG: passing a full URL as key results in a malformed path")
}

func TestDelete_ServerError(t *testing.T) {
	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("storage error"))
	})
	defer ts.Close()

	err := sw.Delete(context.Background(), "some/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

func TestPing_Success(t *testing.T) {
	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	err := sw.Ping(context.Background())
	require.NoError(t, err)
}

func TestPing_Failure(t *testing.T) {
	ts, sw := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer ts.Close()

	err := sw.Ping(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}

func TestNewSeaweedfs_PingFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	_, err := NewSeaweedfs(context.Background(), ts.URL, "", ts.Client())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "seaweedfs connection failed")
}

func TestNewSeaweedfs_PublicURLDefault(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sw, err := NewSeaweedfs(context.Background(), ts.URL, "", ts.Client())
	require.NoError(t, err)
	assert.Equal(t, ts.URL, sw.publicURL, "publicURL should default to baseURL when empty")
}

package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type Seaweedfs struct {
	baseURL   string // internal Docker URL for actual HTTP calls
	publicURL string // URL returned to clients (mobile app)
	client    *http.Client
}

func NewSeaweedfs(ctx context.Context, url string, publicURL string, client *http.Client) (*Seaweedfs, error) {
	pub := publicURL
	if pub == "" {
		pub = url
	}
	s := &Seaweedfs{baseURL: url, publicURL: pub, client: client}

	if err := s.Ping(ctx); err != nil {
		return nil, fmt.Errorf("seaweedfs connection failed: %w", err)
	}
	return s, nil
}

func (s *Seaweedfs) Put(ctx context.Context, key string, r *bytes.Reader) (string, error) {
	internalURL := fmt.Sprintf("%s/%s", s.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, internalURL, r)
	if err != nil {
		return "", fmt.Errorf("create put request: %w", err)
	}

	// determine mime type dynamically or just set generic
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send put request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %s - %s", resp.Status, string(body))
	}

	// return public URL for client-facing responses
	publicURL := fmt.Sprintf("%s/%s", s.publicURL, key)
	return publicURL, nil
}

func (s *Seaweedfs) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/%s", s.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (s *Seaweedfs) Get(ctx context.Context, key string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", s.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending get request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get failed: %s - %s", resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

func (s *Seaweedfs) Ping(ctx context.Context) error {
	// Filer API doesn't have /healthz, use root endpoint
	url := s.baseURL + "/"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending ping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}

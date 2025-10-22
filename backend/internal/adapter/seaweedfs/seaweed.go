package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type Seaweedfs struct {
	baseURL string
	client  *http.Client
}

func NewSeaweedfs(ctx context.Context, url string, client *http.Client) (*Seaweedfs, error) {
	s := &Seaweedfs{baseURL: url, client: client}

	if err := s.Ping(ctx); err != nil {
		return nil, fmt.Errorf("seaweedfs connection failed: %w", err)
	}
	return s, nil
}

func (s *Seaweedfs) Put(ctx context.Context, key string, r *bytes.Reader) (string, error) {
	url := fmt.Sprintf("%s/%s", s.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, r)
	if err != nil {
		return "", fmt.Errorf("creating put request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending put request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %s - %s", resp.Status, string(body))
	}

	return key, nil
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
	url := s.baseURL + "/healthz"

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

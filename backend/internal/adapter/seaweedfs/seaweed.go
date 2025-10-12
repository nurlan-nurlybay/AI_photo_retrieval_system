package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
)

type Seaweedfs struct {
	baseURL string
}

func NewSeaweedfs(url string) (*Seaweedfs, error) {
	s := &Seaweedfs{baseURL: url}

	if err := s.Ping(); err != nil {
		return nil, fmt.Errorf("seaweedfs connection failed: %w", err)
	}
	return s, nil
}

func (s *Seaweedfs) Put(ctx context.Context, key string, r *bytes.Reader) (publicURL string, err error) {
	req, err := http.NewRequest("POST", s.baseURL+"/"+key, r)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return key, nil
}

func (s *Seaweedfs) Delete(ctx context.Context, key string) error {

	return nil
}

func (s *Seaweedfs) Ping() error {
	resp, err := http.Get(s.baseURL + "/dir/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

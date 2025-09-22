package faiss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	faissdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type Client struct {
	baseURL string
	http    *http.Client
}

var _ usecase.VectorIndex = (*Client)(nil)

func NewClient(cfg config.Faiss) usecase.VectorIndex {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port),
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) Insert(ctx context.Context, id string, vector []float32) error {
	req := faissdto.VectorAddRequest{
		ID:        id,
		Vector:    vector,
		Normalize: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal insert request: %w", err)
	}

	url := c.baseURL + "/v1/vectors/add"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build insert request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send insert request: %w", err)
	}
	defer resp.Body.Close()

	var out faissdto.VectorAddResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode insert response: %w", err)
	}
	if !out.Ok {
		return fmt.Errorf("faiss insert failed: %v", out.Error)
	}

	return nil
}

func (c *Client) Search(ctx context.Context, vector []float32, k int) ([]usecase.SearchResult, error) {
	req := faissdto.VectorSearchRequest{
		Vector:    vector,
		K:         k,
		Normalize: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}

	url := c.baseURL + "/v1/vectors/search"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send search request: %w", err)
	}
	defer resp.Body.Close()

	var out faissdto.VectorSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	if !out.Ok {
		return nil, fmt.Errorf("faiss search failed: %v", out.Error)
	}

	results := make([]usecase.SearchResult, len(out.Results))
	for i, r := range out.Results {
		results[i] = usecase.SearchResult{ID: r.ID, Score: r.Score}
	}
	return results, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	req := faissdto.VectorDeleteRequest{ID: id}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal delete request: %w", err)
	}

	url := c.baseURL + "/v1/vectors/delete"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build delete request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send delete request: %w", err)
	}
	defer resp.Body.Close()

	var out faissdto.VectorDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode delete response: %w", err)
	}
	if !out.Ok {
		return fmt.Errorf("faiss delete failed: %v", out.Error)
	}

	return nil
}
package faiss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	faissdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type Client struct {
	baseURL string
	http    *http.Client
}

var _ usecase.VectorIndex = (*Client)(nil)

func NewClient(ctx context.Context, cfg config.Faiss, httpClient *http.Client) (usecase.VectorIndex, error) {
	base := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)

	client := &Client{
		baseURL: base,
		http:    httpClient,
	}

	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clip connection failed: %w", err)
	}

	return client, nil
}

func (c *Client) Insert(ctx context.Context, userID, mediaID int64, vector []float32) error {
	if len(vector) == 0 {
		return fmt.Errorf("cannot insert empty vector for media_id=%d", mediaID)
	}
	namespace := strconv.FormatInt(userID, 10)

	req := faissdto.VectorAddRequest{
		Namespace: namespace,
		ID:        mediaID,
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

	// Check HTTP status before decoding
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("faiss insert HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	// Decode JSON normally
	var out faissdto.VectorAddResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode insert response: %w", err)
	}

	// Safe error dereference
	if !out.OK {
		if out.Error != nil {
			return fmt.Errorf("faiss error: %s", *out.Error)
		}
		return fmt.Errorf("faiss returned not ok")
	}

	// heck for returned dim
	if out.Dim != nil && *out.Dim != len(vector) {
		return fmt.Errorf("dimension mismatch: server expects %d but sent %d", *out.Dim, len(vector))
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
	if !out.OK {
		return nil, fmt.Errorf("faiss search failed: %v", out.Error)
	}

	results := make([]usecase.SearchResult, len(out.Results))
	for i, r := range out.Results {
		results[i] = usecase.SearchResult{ID: r.ID, Score: r.Score}
	}
	return results, nil
}

func (c *Client) Delete(ctx context.Context, id int64) error {
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
	if !out.OK {
		return fmt.Errorf("faiss delete failed: %v", out.Error)
	}

	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/healthz", nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

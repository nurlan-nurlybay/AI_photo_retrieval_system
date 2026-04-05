package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type Config struct {
	URL string
}

type Client struct {
	client *http.Client
	cfg    Config
}

func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		client: httpClient,
		cfg:    cfg,
	}
}

type searchRequest struct {
	Namespace   string    `json:"namespace"`
	QueryText   string    `json:"query_text,omitempty"`
	ImageVector []float32 `json:"image_vector,omitempty"`
	TextVector  []float32 `json:"text_vector,omitempty"`
	TopK        int       `json:"top_k"`
}

type searchResponse struct {
	UsedQwen bool `json:"used_qwen"`
	Results  []struct {
		ID       int64   `json:"id"`
		Distance float32 `json:"distance"` // distance could be inner product (score) or L2
	} `json:"results"`
}

type IngestItem struct {
	ImageID int64       `json:"image_id"`
	Vector  []float32   `json:"vector"`
}

type IngestBatchRequest struct {
	UserID int64        `json:"user_id"`
	Items  []IngestItem `json:"items"`
}

func (c *Client) IngestImageBatch(ctx context.Context, userID int64, items []IngestItem) error {
	return c.sendIngest(ctx, "/v1/ingest/image/batch", userID, items)
}

func (c *Client) IngestTextBatch(ctx context.Context, userID int64, items []IngestItem) error {
	return c.sendIngest(ctx, "/v1/ingest/text/batch", userID, items)
}

func (c *Client) sendIngest(ctx context.Context, path string, userID int64, items []IngestItem) error {
	reqBody := IngestBatchRequest{
		UserID: userID,
		Items:  items,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal vector ingest request: %w", err)
	}

	endpoint := fmt.Sprintf("%s%s", c.cfg.URL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create ingest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("vector service http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vector service ingest returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *Client) DeleteImage(ctx context.Context, namespace string, imageID int64) error {
	endpoint := fmt.Sprintf("%s/v1/delete?namespace=%s&image_id=%d", c.cfg.URL, namespace, imageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("vector service http error on delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// ignoring 404s since deletion may happen on partial ingested files
		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("vector service delete returned status %d: %s", resp.StatusCode, string(bodyBytes))
		}
	}
	return nil
}

func (c *Client) SearchHybrid(ctx context.Context, namespace, queryText string, imageVec []float32, textVec []float32, topK int) ([]usecase.SearchResult, bool, error) {
	reqBody := searchRequest{
		Namespace:   namespace,
		QueryText:   queryText,
		ImageVector: imageVec,
		TextVector:  textVec,
		TopK:        topK,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal vector search request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/search/hybrid", c.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("vector service http error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("vector service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var sr searchResponse
	if err := json.Unmarshal(bodyBytes, &sr); err != nil {
		return nil, false, fmt.Errorf("failed to parse vector search response: %w", err)
	}

	out := make([]usecase.SearchResult, 0, len(sr.Results))
	for _, raw := range sr.Results {
		out = append(out, usecase.SearchResult{
			ID:    raw.ID,
			Score: raw.Distance,
		})
	}

	return out, sr.UsedQwen, nil
}

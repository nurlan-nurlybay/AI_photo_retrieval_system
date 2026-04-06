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

func UserNamespace(userID int64) string {
	return fmt.Sprintf("user_%d", userID)
}

// --- Ingest request/response types (match Python vector service models) ---

type imageVectorItem struct {
	ID          int64     `json:"id"`
	ImageVector []float32 `json:"image_vector"`
}

type addImageBatchRequest struct {
	Namespace string            `json:"namespace"`
	Items     []imageVectorItem `json:"items"`
}

type textVectorItem struct {
	ID         int64     `json:"id"`
	TextVector []float32 `json:"text_vector"`
	Tags       []string  `json:"tags"`
}

type addTextBatchRequest struct {
	Namespace string           `json:"namespace"`
	Items     []textVectorItem `json:"items"`
}

// IngestItem is the public type used by the worker to pass data.
type IngestItem struct {
	ImageID int64
	Vector  []float32
	Tags    []string
}

func (c *Client) IngestImageBatch(ctx context.Context, userID int64, items []IngestItem) error {
	reqItems := make([]imageVectorItem, len(items))
	for i, it := range items {
		reqItems[i] = imageVectorItem{ID: it.ImageID, ImageVector: it.Vector}
	}
	body := addImageBatchRequest{
		Namespace: UserNamespace(userID),
		Items:     reqItems,
	}
	return c.postJSON(ctx, "/v1/ingest/image", body)
}

func (c *Client) IngestTextBatch(ctx context.Context, userID int64, items []IngestItem) error {
	reqItems := make([]textVectorItem, len(items))
	for i, it := range items {
		reqItems[i] = textVectorItem{ID: it.ImageID, TextVector: it.Vector, Tags: it.Tags}
	}
	body := addTextBatchRequest{
		Namespace: UserNamespace(userID),
		Items:     reqItems,
	}
	return c.postJSON(ctx, "/v1/ingest/text", body)
}

// --- Deletion types (match Python vector service endpoints) ---

type deleteItemsRequest struct {
	Namespace string  `json:"namespace"`
	ImageIDs  []int64 `json:"image_ids"`
}

func (c *Client) DeleteItems(ctx context.Context, userID int64, imageIDs []int64) error {
	if len(imageIDs) == 0 {
		return nil
	}
	body := deleteItemsRequest{
		Namespace: UserNamespace(userID),
		ImageIDs:  imageIDs,
	}
	return c.postJSON(ctx, "/v1/delete/items", body)
}

func (c *Client) ClearNamespace(ctx context.Context, userID int64) error {
	path := fmt.Sprintf("/v1/admin/clear/%s", UserNamespace(userID))
	return c.postJSON(ctx, path, nil)
}

func (c *Client) NukeNamespace(ctx context.Context, userID int64) error {
	// For per-user account deletion, we clear only that user's namespace.
	// The global /v1/admin/nuke endpoint is reserved for admin-only use.
	path := fmt.Sprintf("/v1/admin/clear/%s", UserNamespace(userID))
	return c.postJSON(ctx, path, nil)
}

// --- Search types (match Python SearchRequest / response) ---

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
		Distance float32 `json:"distance"`
	} `json:"results"`
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

// --- Helpers ---

func (c *Client) postJSON(ctx context.Context, path string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.cfg.URL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("vector service http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vector service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

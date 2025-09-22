package clip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *Client) EmbedText(ctx context.Context, text string) ([]float32, error) {
	payload, _ := json.Marshal(map[string]string{"text": text})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed/text", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clip error: %s", string(body))
	}

	var result embedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

func (c *Client) EmbedImage(ctx context.Context, data []byte) ([]float32, error) {
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed/image", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clip error: %s", string(body))
	}

	var result embedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

package clip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(ctx context.Context, cfg config.Clip, httpClent *http.Client) (usecase.Embedder, error) {
	base := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)

	client := &Client{
		baseURL: base,
		http:    httpClent,
	}

	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clip connection failed: %w", err)
	}
	return client, nil
}

func (c *Client) EmbedText(ctx context.Context, text string) ([]float64, error) {
	reqBody := map[string]string{"text": text}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.http.Post(c.baseURL+"/v1/encode/text", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python service returned %s", resp.Status)
	}

	var respBody struct {
		Vector []float64 `json:"vector"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(respBody.Vector) != 512 {
		return nil, errors.New("invalid vector length")
	}

	return respBody.Vector, nil

}

func (c *Client) EmbedImage(ctx context.Context, data []byte, filename string) ([]float64, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/encode/image", &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python service returned %s", resp.Status)
	}

	var respBody struct {
		Vector []float64 `json:"vector"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(respBody.Vector) != 512 {
		return nil, errors.New("invalid vector length")
	}

	return respBody.Vector, nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ping", nil)
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

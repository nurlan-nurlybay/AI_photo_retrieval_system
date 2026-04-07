package clip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClientFromURL(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, http: httpClient}
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

func (c *Client) EmbedText(ctx context.Context, text string) ([]float32, error) {
	url := c.baseURL + "/v1/encode/text/"

	reqBody := clipdto.TextRequest{
		Texts: []string{text},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create req failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post req failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clip service returned %s: %s", resp.Status, string(bodyBytes))
	}

	var respBody clipdto.VectorResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(respBody.Vectors) == 0 || len(respBody.Vectors[0]) == 0 {
		return nil, fmt.Errorf("empty vector response")
	}

	return respBody.Vectors[0], nil
}

func (c *Client) EmbedImage(ctx context.Context, data []byte) ([]float32, error) {
	url := c.baseURL + "/v1/encode/image/fast/"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("files", "image.jpg")
	if err != nil {
		return nil, fmt.Errorf("create form file failed: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return nil, fmt.Errorf("write image bytes failed: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &b)
	if err != nil {
		return nil, fmt.Errorf("create req failed: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post req failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clip service returned %s: %s", resp.Status, string(bodyBytes))
	}

	var respBody clipdto.VectorResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(respBody.Vectors) == 0 || len(respBody.Vectors[0]) == 0 {
		return nil, fmt.Errorf("empty vector response")
	}

	return respBody.Vectors[0], nil
}

func (c *Client) EmbedImageURL(ctx context.Context, imgUrl string) ([]float32, error) {
	endpoint := c.baseURL + "/v1/encode/image/url/fast/"

	reqBody := map[string]interface{}{
		"urls": []string{imgUrl},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create req failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post req failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clip service returned %s: %s", resp.Status, string(bodyBytes))
	}

	var respBody clipdto.VectorResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(respBody.Vectors) == 0 || len(respBody.Vectors[0]) == 0 {
		return nil, fmt.Errorf("empty vector response")
	}

	return respBody.Vectors[0], nil
}

func (c *Client) EmbedImageURLSlow(ctx context.Context, imgUrl string) (*clipdto.SlowEncodeResult, error) {
	endpoint := c.baseURL + "/v1/encode/image/url/slow/"

	reqBody := map[string]interface{}{
		"urls": []string{imgUrl},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create req failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post req failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clip service returned %s: %s", resp.Status, string(bodyBytes))
	}

	var respBody clipdto.SlowEncodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(respBody.Results) == 0 {
		return nil, fmt.Errorf("empty slow response")
	}

	return &respBody.Results[0], nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
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

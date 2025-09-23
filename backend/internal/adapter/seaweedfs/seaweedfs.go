package seaweedfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Client struct {
	MasterURL string // http://localhost:9333
	VolumeURL string // optional, can be discovered dynamically
}

type assignResponse struct {
	Fid       string `json:"fid"`
	URL       string `json:"url"`
	PublicURL string `json:"publicUrl"`
	Count     int    `json:"count"`
}

// Returns public URL
func (c *Client) Upload(filePath string) (string, error) {
	// ask master for file id
	resp, err := http.Get(fmt.Sprintf("%s/dir/assign", c.MasterURL))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var assign assignResponse
	if err := json.NewDecoder(resp.Body).Decode(&assign); err != nil {
		return "", err
	}

	// upload file to volume server
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	uploadURL := fmt.Sprintf("http://%s/%s", assign.URL, assign.Fid)
	req, err := http.NewRequest("POST", uploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	upResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer upResp.Body.Close()

	if upResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(upResp.Body)
		return "", fmt.Errorf("upload failed: %s", string(b))
	}

	return fmt.Sprintf("http://%s/%s", assign.PublicURL, assign.Fid), nil
}

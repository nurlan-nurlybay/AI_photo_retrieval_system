package faiss

import (
    "context"
    "fmt"
)

type Client struct {
    host string
    port int
}

func NewClient(host string, port int) (*Client, error) {
    return &Client{
        host: host,
        port: port,
    }, nil
}

func (c *Client) Search(ctx context.Context, deviceID string, embedding []float32, k int) ([]string, error) {
    fmt.Printf("Searching in FAISS at %s:%d for device=%s, k=%d\n", c.host, c.port, deviceID, k)
    return []string{}, nil
}

func (c *Client) Insert(ctx context.Context, deviceID, id string, embedding []float32) error {
    return nil
}

func (c *Client) Delete(ctx context.Context, deviceID, id string) error {
    return nil
}

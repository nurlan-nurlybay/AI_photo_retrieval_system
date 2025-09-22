package domain

import "time"

type Media struct {
	ID        string
	DeviceID  string
	URL       string
	ThumbURL  string
	CreatedAt time.Time
	Deleted   bool
}

func (m Media) IsActive() bool {
	return !m.Deleted
}

type MediaRepo interface {
	FindByIDs(deviceID string, ids []string) ([]Media, error)
}

type Embedder interface {
	EmbedText(text string) ([]float32, error)
	EmbedImage(data []byte) ([]float32, error)
}

type VectorIndex interface {
	Search(deviceID string, embedding []float32, k int) ([]string, error)
}

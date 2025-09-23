// adapter/faiss/faissdto/models.go
package faissdto

type VectorAddRequest struct {
	ID        string    `json:"id"`
	Vector    []float32 `json:"vector"`
	Normalize bool      `json:"normalize"`
}

type VectorAddResponse struct {
	Ok       bool    `json:"ok"`
	ID       string  `json:"id"`
	Replaced bool    `json:"replaced"`
	Dim      *int    `json:"dim,omitempty"`
	Error    *string `json:"error,omitempty"`
}

type VectorDeleteRequest struct {
	ID string `json:"id"`
}

type VectorDeleteResponse struct {
	Ok      bool    `json:"ok"`
	ID      *string `json:"id,omitempty"`
	Deleted *bool   `json:"deleted,omitempty"`
	Error   *string `json:"error,omitempty"`
}

type VectorSearchRequest struct {
	Vector    []float32 `json:"vector"`
	K         int       `json:"k"`
	Normalize bool      `json:"normalize"`
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

type VectorSearchResponse struct {
	Ok       bool           `json:"ok"`
	K        int            `json:"k"`
	Results  []SearchResult `json:"results"`
	Degraded bool           `json:"degraded"`
	TookMs   *int           `json:"tookMs,omitempty"`
	Error    *string        `json:"error,omitempty"`
}

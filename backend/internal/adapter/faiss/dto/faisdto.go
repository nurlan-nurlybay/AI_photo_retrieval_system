package faissdto

type VectorAddRequest struct {
	ID        int64     `json:"id"`
	Vector    []float64 `json:"vector"`
	Normalize bool      `json:"normalize"`
}

type VectorAddResponse struct {
	Ok       bool    `json:"ok"`
	ID       int64   `json:"id"`
	Replaced bool    `json:"replaced"`
	Dim      *int    `json:"dim,omitempty"`
	Error    *string `json:"error,omitempty"`
}

type VectorDeleteRequest struct {
	ID int64 `json:"id"`
}

type VectorDeleteResponse struct {
	Ok      bool    `json:"ok"`
	ID      *int64  `json:"id,omitempty"`
	Deleted *bool   `json:"deleted,omitempty"`
	Error   *string `json:"error,omitempty"`
}

type VectorSearchRequest struct {
	Vector    []float64 `json:"vector"`
	K         int       `json:"k"`
	Normalize bool      `json:"normalize"`
}

type SearchResult struct {
	ID    int64   `json:"id"`
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

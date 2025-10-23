package faissdto

import "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"

// ---------- Add ----------
type VectorAddRequest struct {
	Namespace string    `json:"namespace"`
	ID        int64     `json:"id"`
	Vector    []float32 `json:"vector"`
	Normalize bool      `json:"normalize"`
}

type VectorAddResponse struct {
	OK        bool    `json:"ok"`
	ID        int64   `json:"id"`
	Namespace string  `json:"namespace"`
	Replaced  bool    `json:"replaced"`
	Dim       *int    `json:"dim,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// ---------- Delete ----------
type VectorDeleteRequest struct {
	Namespace *string `json:"namespace,omitempty"`
	Model     *string `json:"model,omitempty"`
	ID        int64   `json:"id"`
}

type VectorDeleteResponse struct {
	OK        bool    `json:"ok"`
	ID        *int64  `json:"id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Deleted   *bool   `json:"deleted,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// ---------- Search ----------
type VectorSearchRequest struct {
	Namespace *string   `json:"namespace,omitempty"`
	Model     *string   `json:"model,omitempty"`
	Vector    []float32 `json:"vector"`
	K         int       `json:"k"`
	Normalize bool      `json:"normalize"`
}

type VectorSearchResponse struct {
	OK        bool                   `json:"ok"`
	Namespace string                 `json:"namespace"`
	K         int                    `json:"k"`
	Results   []usecase.SearchResult `json:"results"`
	Degraded  bool                   `json:"degraded"`
	TookMs    *int                   `json:"tookMs,omitempty"`
	Error     *string                `json:"error,omitempty"`
}

// ---------- Health ----------
type HealthResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

package clipdto

type TextRequest struct {
	Text string `json:"text"`
}

type VectorResponse struct {
	Vector []float64 `json:"vector"`
}

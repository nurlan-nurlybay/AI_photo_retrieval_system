package clipdto

type TextRequest struct {
	Texts []string `json:"texts"`
}

type VectorResponse struct {
	Vectors [][]float32 `json:"vectors"`
}

type SlowEncodeResult struct {
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	TextVector  []float32 `json:"text_vector"`
}

type SlowEncodeResponse struct {
	Results []SlowEncodeResult `json:"results"`
}

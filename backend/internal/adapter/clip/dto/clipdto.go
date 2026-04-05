package clipdto

type TextRequest struct {
	Texts []string `json:"texts"`
}

type VectorResponse struct {
	Vectors [][]float32 `json:"vectors"`
}

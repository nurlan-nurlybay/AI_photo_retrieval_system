package clipdto

type EncodeTextRequest struct {
	Req     TextRequest   `json:"req"`
	Options EncodeOptions `json:"options"`
}

type EncodeOptions struct {
	Model     string `json:"model"`
	Normalize bool   `json:"normalize"`
	Quantize  bool   `json:"quantize"`
}

type TextRequest struct {
	Texts []string `json:"texts"`
}

type VectorResponse struct {
	Vectors [][]float32 `json:"vectors"`
}

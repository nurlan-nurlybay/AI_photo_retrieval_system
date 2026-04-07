package clipdto

<<<<<<< HEAD
type EncodeTextRequest struct {
	Req     TextRequest   `json:"req"`
	Options EncodeOptions `json:"options"`
}

type EncodeOptions struct {
	Model     string `json:"model"`
	Normalize bool   `json:"normalize"`
	Quantize  bool   `json:"quantize"`
}

=======
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
type TextRequest struct {
	Texts []string `json:"texts"`
}

type VectorResponse struct {
	Vectors [][]float32 `json:"vectors"`
}
<<<<<<< HEAD
=======

type SlowEncodeResult struct {
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	TextVector  []float32 `json:"text_vector"`
}

type SlowEncodeResponse struct {
	Results []SlowEncodeResult `json:"results"`
}
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1

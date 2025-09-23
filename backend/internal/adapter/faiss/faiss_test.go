package faiss_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

const baseURL = "http://localhost:8002/v1/vectors"

func TestFaissIntegration(t *testing.T) {
	// Insert media_1
	vec1 := map[string]any{
		"id":     "media_1",
		"vector": []float32{0.1, 0.2, 0.3, 0.4, 0.5},
	}
	if err := postJSON(baseURL+"/add", vec1); err != nil {
		t.Fatalf("insert media_1 failed: %v", err)
	}
	t.Log("Inserted media_1")

	// Insert media_2
	vec2 := map[string]any{
		"id":     "media_2",
		"vector": []float32{0.11, 0.19, 0.29, 0.41, 0.52},
	}
	if err := postJSON(baseURL+"/add", vec2); err != nil {
		t.Fatalf("insert media_2 failed: %v", err)
	}
	t.Log("Inserted media_2")

	// Search should return media_1 first
	results := searchVectors(t, []float32{0.1, 0.2, 0.3, 0.4, 0.5}, 2)
	t.Logf("Initial search results: %+v", results)
	if results[0].ID != "media_1" {
		t.Errorf("expected media_1, got %q", results[0].ID)
	}

	// Delete media_1
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/media_1", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on delete, got %d", resp.StatusCode)
	}
	t.Log("Deleted media_1")

	// Search again → top hit should not be media_1
	results = searchVectors(t, []float32{0.1, 0.2, 0.3, 0.4, 0.5}, 2)
	t.Logf("Post-delete search results: %+v", results)
	if results[0].ID == "media_1" {
		t.Errorf("expected media_1 gone, got %q", results[0].ID)
	}

	// Insert duplicate ID media_2 with new vector
	vec2Update := map[string]any{
		"id":     "media_2",
		"vector": []float32{0.9, 0.9, 0.9, 0.9, 0.9},
	}
	if err := postJSON(baseURL+"/add", vec2Update); err != nil {
		t.Fatalf("duplicate insert failed: %v", err)
	}
	t.Log("Updated media_2 with new vector")

	// Search near [0.9 …] → should hit media_2
	results = searchVectors(t, []float32{0.9, 0.9, 0.9, 0.9, 0.9}, 1)
	t.Logf("Duplicate update search results: %+v", results)
	if results[0].ID != "media_2" {
		t.Errorf("expected media_2, got %q", results[0].ID)
	}

	// Garbage input (missing vector)
	badReq := map[string]any{"id": "oops"}
	err = postJSON(baseURL+"/add", badReq)
	if err == nil {
		t.Error("expected error for garbage input, got nil")
	} else {
		t.Logf("Garbage input correctly failed: %v", err)
	}
}

// --- helpers ---

func postJSON(url string, body map[string]any) error {
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &httpError{code: resp.StatusCode}
	}
	return nil
}

func postJSONResp(url string, body map[string]any, out any) error {
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

func searchVectors(t *testing.T, vec []float32, k int) []SearchResult {
	req := map[string]any{"vector": vec, "k": k}
	var resp struct {
		Results []SearchResult `json:"results"`
	}
	if err := postJSONResp(baseURL+"/search", req, &resp); err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected results, got none")
	}
	return resp.Results
}

type httpError struct {
	code int
}

func (e *httpError) Error() string {
	return http.StatusText(e.code)
}

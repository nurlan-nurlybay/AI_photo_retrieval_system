package faiss_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	faissdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss/dto"
)

const baseURL = "http://localhost:8002/v1/vectors"

func TestFaissIntegration(t *testing.T) {
	// quick health check
	resp, err := http.Get("http://localhost:8002/v1/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Skip("FAISS service not running on :8002, skipping integration test")
	}

	// Insert media_1
	vec1 := faissdto.VectorAddRequest{
		ID:        1,
		Vector:    makeVec(0.1),
		Normalize: true,
	}
	if err := postJSON(baseURL+"/add", vec1); err != nil {
		t.Fatalf("insert media_1 failed: %v", err)
	}
	t.Log("Inserted media_1")

	// Insert media_2
	vec2 := faissdto.VectorAddRequest{
		ID:        2,
		Vector:    makeVec(0.2),
		Normalize: true,
	}
	if err := postJSON(baseURL+"/add", vec2); err != nil {
		t.Fatalf("insert media_2 failed: %v", err)
	}
	t.Log("Inserted media_2")

	// Search should return media_1 first
	results := searchVectors(t, makeVec(0.1), 2)
	t.Logf("Initial search results: %+v", results)
	if results[0].ID != 1 {
		t.Errorf("expected media_1, got %q", results[0].ID)
	}

	// Delete media_1
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/1", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on delete, got %d", resp.StatusCode)
	}
	t.Log("Deleted media_1")

	// Search again → top hit should not be media_1
	results = searchVectors(t, makeVec(0.1), 2)
	t.Logf("Post-delete search results: %+v", results)
	if results[0].ID == 1 {
		t.Errorf("expected media_1 gone, got %q", results[0].ID)
	}

	// Insert duplicate ID media_2 with new vector
	vec2Update := faissdto.VectorAddRequest{
		ID:        2,
		Vector:    makeVec(0.9),
		Normalize: true,
	}
	if err := postJSON(baseURL+"/add", vec2Update); err != nil {
		t.Fatalf("duplicate insert failed: %v", err)
	}
	t.Log("Updated media_2 with new vector")

	// Search near [0.9 …] → should hit media_2
	results = searchVectors(t, makeVec(0.9), 1)
	t.Logf("Duplicate update search results: %+v", results)
	if results[0].ID != 2 {
		t.Errorf("expected media_2, got %q", results[0].ID)
	}

	// Garbage input (missing vector)
	badReq := faissdto.VectorAddRequest{ID: 1234}
	err = postJSON(baseURL+"/add", badReq)
	if err == nil {
		t.Error("expected error for garbage input, got nil")
	} else {
		t.Logf("Garbage input correctly failed: %v", err)
	}
}

// --- helpers ---

func postJSON(url string, body any) error {
	b, _ := json.MarshalIndent(body, "", "  ")
	fmt.Printf(">>> OUTGOING JSON:\n%s\n", string(b)) // 👈 sanity log
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func postJSONResp(url string, body any, out any) error {
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

type SearchResult struct {
	ID    int64   `json:"id"`
	Score float32 `json:"score"`
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

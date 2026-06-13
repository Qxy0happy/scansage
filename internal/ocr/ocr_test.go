package ocr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOCRPage(t *testing.T) {
	var pathOK = true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			pathOK = false
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "# Mock OCR Result\n\nThis is page 1.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	pngBytes := []byte("fake-png-data")
	result, err := OCRPage(server.URL, "", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !pathOK {
		t.Fatal("unexpected request path")
	}
	if result != "# Mock OCR Result\n\nThis is page 1." {
		t.Fatalf("unexpected result: %s", result)
	}
}

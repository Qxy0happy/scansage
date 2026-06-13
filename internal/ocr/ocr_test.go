package ocr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOCRPage(t *testing.T) {
	var (
		pathOK   = true
		authSeen string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			pathOK = false
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		authSeen = r.Header.Get("Authorization")
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

	t.Run("no api key", func(t *testing.T) {
		result, err := OCRPage(server.URL, "", "", pngBytes)
		if err != nil {
			t.Fatal(err)
		}
		if !pathOK {
			t.Fatal("unexpected request path")
		}
		if result != "# Mock OCR Result\n\nThis is page 1." {
			t.Fatalf("unexpected result: %s", result)
		}
	})

	t.Run("with api key", func(t *testing.T) {
		authSeen = ""
		result, err := OCRPage(server.URL, "sk-test123", "my-model", pngBytes)
		if err != nil {
			t.Fatal(err)
		}
		if authSeen != "Bearer sk-test123" {
			t.Fatalf("expected Bearer sk-test123, got %q", authSeen)
		}
		if result != "# Mock OCR Result\n\nThis is page 1." {
			t.Fatalf("unexpected result: %s", result)
		}
	})
}

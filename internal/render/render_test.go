package render

import (
	"path/filepath"
	"testing"
)

func TestRenderPages(t *testing.T) {
	pdfPath, err := filepath.Abs(filepath.Join("..", "..", "testdata", "test.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	pages, err := RenderAll(pdfPath, 300)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("expected at least 1 page")
	}
	for i, png := range pages {
		if len(png) == 0 {
			t.Fatalf("page %d: empty PNG bytes", i)
		}
		t.Logf("page %d: %d bytes", i+1, len(png))
	}
}

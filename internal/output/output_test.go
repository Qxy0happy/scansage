package output

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWritePages(t *testing.T) {
	tmpDir := t.TempDir()
	pages := []string{
		"# Page 1\n\nContent",
		"# Page 2\n\nMore content",
	}

	err := WritePages(tmpDir, pages)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(pages); i++ {
		name := fmt.Sprintf("page_%03d.md", i+1)
		path := filepath.Join(tmpDir, "pages", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != pages[i] {
			t.Fatalf("%s: got %q, want %q", name, string(data), pages[i])
		}
	}
}

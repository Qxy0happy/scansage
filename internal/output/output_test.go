package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritePage(t *testing.T) {
	tmp := t.TempDir()
	err := WritePage(tmp, 0, "# Page 1")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tmp, "pages", "page_001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Page 1" {
		t.Fatalf("got %q", string(data))
	}
}

func TestHasPage(t *testing.T) {
	tmp := t.TempDir()
	if HasPage(tmp, 0) {
		t.Fatal("should not exist yet")
	}
	WritePage(tmp, 1, "hello")
	if !HasPage(tmp, 1) {
		t.Fatal("should exist")
	}
}

func TestLastPageIndex(t *testing.T) {
	tmp := t.TempDir()

	// empty dir
	if i := LastPageIndex(tmp); i != -1 {
		t.Fatalf("expected -1, got %d", i)
	}

	// write pages 1 and 3 (skip 2)
	WritePage(tmp, 0, "a")
	WritePage(tmp, 2, "c")

	// last is 3rd page → 0-indexed 2
	if i := LastPageIndex(tmp); i != 2 {
		t.Fatalf("expected 2, got %d", i)
	}
}

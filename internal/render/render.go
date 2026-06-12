package render

import (
	"fmt"

	"github.com/gen2brain/go-fitz"
)

func RenderAll(path string, dpi float64) ([][]byte, error) {
	doc, err := fitz.New(path)
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	defer doc.Close()

	n := doc.NumPage()
	pages := make([][]byte, 0, n)

	for i := 0; i < n; i++ {
		png, err := doc.ImagePNG(i, dpi)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i, err)
		}
		pages = append(pages, png)
	}

	return pages, nil
}

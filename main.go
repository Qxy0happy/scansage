package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/Qxy0happy/scansage/internal/ocr"
	"github.com/Qxy0happy/scansage/internal/output"
	"github.com/Qxy0happy/scansage/internal/render"
)

func main() {
	cmd := &cli.Command{
		Name:  "scansage",
		Usage: "PDF → PNG → GLM-OCR → raw markdown pages",
		UsageText: `scansage <input.pdf> [-o <dir>] [--ocr-url <url>] [--dpi <n>]

Examples:
  scansage mydoc.pdf
  scansage mydoc.pdf -o ./output
  scansage mydoc.pdf --ocr-url http://192.168.1.100:8080`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   ".",
				Usage:   "output directory (creates pages/ inside)",
			},
			&cli.StringFlag{
				Name:  "ocr-url",
				Value: "http://localhost:8080",
				Usage: "llama.cpp OpenAI-compatible API URL",
			},
			&cli.FloatFlag{
				Name:  "dpi",
				Value: 300,
				Usage: "PDF rendering DPI",
			},
			&cli.IntFlag{
				Name:  "concurrency",
				Value: 1,
				Usage: "number of concurrent OCR requests (not yet implemented)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			input := cmd.Args().First()
			if input == "" {
				return fmt.Errorf("usage: scansage <input.pdf> [-o <dir>] [--ocr-url <url>]")
			}

			outDir := cmd.String("output")
			ocrURL := cmd.String("ocr-url")
			dpi := cmd.Float("dpi")

			// Step 1: Render
			log.Printf("rendering %s at %.0f DPI ...", input, dpi)
			pages, err := render.RenderAll(input, dpi)
			if err != nil {
				return fmt.Errorf("render: %w", err)
			}
			log.Printf("rendered %d pages", len(pages))

			// Step 2: OCR each page (serial)
			results := make([]string, len(pages))
			for i, png := range pages {
				log.Printf("OCR page %d/%d ...", i+1, len(pages))
				result, err := ocr.OCRPage(ocrURL, png)
				if err != nil {
					return fmt.Errorf("ocr page %d: %w", i+1, err)
				}
				results[i] = result
			}
			log.Printf("OCR complete for %d pages", len(pages))

			// Step 3: Write output
			if err := output.WritePages(outDir, results); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			log.Printf("output written to %s/pages/", outDir)

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

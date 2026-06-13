package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/Qxy0happy/scansage/internal/ocr"
	"github.com/Qxy0happy/scansage/internal/output"
	"github.com/Qxy0happy/scansage/internal/queue"
	"github.com/Qxy0happy/scansage/internal/render"
	"github.com/Qxy0happy/scansage/internal/skill"
)

func main() {
	cmd := &cli.Command{
		Name:  "scansage",
		Usage: "PDF → PNG → GLM-OCR → raw markdown pages",
		UsageText: `scansage <input.pdf> [-o <dir>] [--ocr-url <url>] [--api-key <key>] [--model <name>] [--dpi <dpi>] [-n <num>]
  scansage skill install <user/repo>
  scansage skill list
  scansage skill run <name> -d <dir>

Examples:
  scansage mydoc.pdf
  scansage mydoc.pdf -o ./output
  scansage mydoc.pdf --ocr-url http://192.168.1.100:8080
  scansage mydoc.pdf --model qwen3-vl
  scansage mydoc.pdf --api-key sk-xxx
  set SCANSAGE_API_KEY=sk-xxx && scansage mydoc.pdf
  scansage skill install Qxy0happy/scansage-skill-refine
  scansage skill run refine -d ./output`,
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
			&cli.StringFlag{
				Name:  "api-key",
				Value: "",
				Usage: "API key for llama.cpp (or set SCANSAGE_API_KEY env var)",
			},
			&cli.StringFlag{
				Name:  "model",
				Value: "",
				Usage: "model name passed to llama.cpp (default: GLM-OCR)",
			},
			&cli.FloatFlag{
				Name:  "dpi",
				Value: 300,
				Usage: "PDF rendering DPI",
			},
			&cli.IntFlag{
				Name:    "concurrency-number",
				Aliases: []string{"n"},
				Value:   1,
				Usage:   "number of concurrent OCR workers (default 1 = serial)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			input := cmd.Args().First()
			if input == "" {
				return fmt.Errorf("usage: scansage <input.pdf> [-o <dir>] [--ocr-url <url>]")
			}

			outDir := cmd.String("output")
			ocrURL := cmd.String("ocr-url")
			apiKey := cmd.String("api-key")
			if apiKey == "" {
				apiKey = os.Getenv("SCANSAGE_API_KEY")
			}
			model := cmd.String("model")
			dpi := cmd.Float("dpi")
			concurrency := cmd.Int("concurrency-number")

			log.Printf("rendering %s at %.0f DPI ...", input, dpi)
			pngs, err := render.RenderAll(input, dpi)
			if err != nil {
				return fmt.Errorf("render: %w", err)
			}
			log.Printf("rendered %d pages", len(pngs))

			lastIdx := output.LastPageIndex(outDir)
			skipTo := lastIdx + 1
			if lastIdx >= 0 {
				log.Printf("resuming from page %d (found %d completed pages)", skipTo+1, lastIdx+1)
			}

			q := queue.New(concurrency, func(job queue.Job) (interface{}, error) {
				i := job.Index
				if i < skipTo {
					return nil, nil
				}
				log.Printf("OCR page %d/%d ...", i+1, len(pngs))
				result, err := ocr.OCRPage(ocrURL, apiKey, model, job.Data.([]byte))
				if err != nil {
					return nil, fmt.Errorf("page %d: %w", i+1, err)
				}
				if err := output.WritePage(outDir, i, result); err != nil {
					return nil, fmt.Errorf("page %d write: %w", i+1, err)
				}
				return nil, nil
			})

			start := time.Now()
			total := 0
			for i, png := range pngs {
				if i < skipTo {
					continue
				}
				q.Add(queue.Job{Index: i, Data: png})
				total++
			}

			results := q.Wait()
			var errs []string
			for _, r := range results {
				if r.Err != nil {
					errs = append(errs, r.Err.Error())
				}
			}
			elapsed := time.Since(start).Round(time.Second)

			if len(errs) > 0 {
				log.Printf("completed %d/%d pages (%d failed, %s)", total-len(errs), total, len(errs), elapsed)
				for _, e := range errs {
					log.Printf("  error: %s", e)
				}
				return fmt.Errorf("processing incomplete, re-run to resume from failed pages")
			}

			log.Printf("OCR complete: %d pages in %s", total, elapsed)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "skill",
				Usage: "manage post-processing skills",
				UsageText: `scansage skill install <user/repo>
  scansage skill list
  scansage skill run <name> -d <dir>

Examples:
  scansage skill install Qxy0happy/scansage-skill-refine
  scansage skill list
  scansage skill run refine -d ./output`,

				Commands: []*cli.Command{
					{
						Name:      "install",
						Usage:     "install a skill from GitHub releases",
						ArgsUsage: "<user/repo>",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							repo := cmd.Args().First()
							if repo == "" {
								return fmt.Errorf("usage: scansage skill install <user/repo>")
							}
							return skill.Install(repo)
						},
					},
					{
						Name:  "list",
						Usage: "list installed skills",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							skills, err := skill.List()
							if err != nil {
								return err
							}
							if len(skills) == 0 {
								fmt.Println("no skills installed")
								return nil
							}
							fmt.Println("installed skills:")
							for _, s := range skills {
								fmt.Printf("  %s\n", s.Name)
							}
							return nil
						},
					},
					{
						Name:      "run",
						Usage:     "run a skill on output directory",
						ArgsUsage: "<name>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dir",
								Aliases:  []string{"d"},
								Usage:    "output directory (with pages/ inside)",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							name := cmd.Args().First()
							if name == "" {
								return fmt.Errorf("usage: scansage skill run <name> -d <dir>")
							}
							return skill.Run(name, cmd.String("dir"))
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

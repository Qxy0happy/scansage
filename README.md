# ScanSage

PDF → PNG → GLM-OCR → raw markdown pages.

## Install

```bash
go install github.com/Qxy0happy/scansage@v1.0.0
```

Requires CGo and MuPDF (bundled with [go-fitz](https://github.com/gen2brain/go-fitz)).

## Usage

```bash
# 1. Start llama.cpp with GLM-OCR
llama-server -hf ggml-org/GLM-OCR-GGUF

# 2. Run scansage
scansage input.pdf
scansage input.pdf -o ./output
scansage input.pdf --ocr-url http://192.168.1.100:8080
```

Output: `output/pages/page_NNN.md` (one file per page, raw OCR markdown).

## How it works

1. **Render** — go-fitz renders each PDF page as PNG (in memory, 300 DPI)
2. **OCR** — sends each PNG to llama.cpp GLM-OCR via OpenAI-compatible API
3. **Output** — writes raw markdown to `pages/page_NNN.md`

Post-processing (LLM cleanup, summarization, layout) is done externally by your own agent.

## License

AGPL-3.0

# ScanSage — PDF OCR Markdown 工具

## Overview

`scansage` — CLI 工具，将 PDF 每页渲染为 PNG，通过 llama.cpp 的 OpenAI-compatible API 调用 GLM-OCR 视觉模型，输出原始 Markdown。

用户自行管理 llama.cpp sidecar（提供 GLM-OCR GGUF），scansage 只通过 HTTP 调用。后续整理（LLM 编辑、摘要、排版）由用户在工具外自行调度 Agent。

## Tech Stack

- **语言:** Go 1.26.4
- **CLI:** `github.com/urfave/cli/v3`
- **PDF 渲染:** `github.com/gen2brain/go-fitz` (MuPDF CGo binding)
- **无配置文件、无环境变量**

## CLI Interface

```
scansage input.pdf [-o <dir>] [--ocr-url <url>] [--dpi <n>] [--concurrency <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `input.pdf` | (required) | 输入的 PDF 文件 |
| `-o, --output` | `.` | 输出目录，下建 `pages/` |
| `--ocr-url` | `http://localhost:8080` | llama.cpp API 地址 |
| `--dpi` | `300` | PDF 渲染分辨率 |
| `--concurrency` | `1` | OCR 并发数 |

## Pipeline

```
input.pdf → [go-fitz 逐页渲染 PNG (纯内存)]
         → [POST /v1/chat/completions → llama.cpp GLM-OCR]
         → 原始 markdown → output/pages/page_NNN.md
```

## GLM-OCR API Call

```json
POST {ocr-url}/v1/chat/completions
{
  "messages": [{
    "role": "user",
    "content": [
      {"type": "image_url", "image_url": {"url": "data:image/png;base64,<PNG bytes>"}},
      {"type": "text", "text": "OCR markdown"}
    ]
  }]
}
```

建议温度 0.1（llama.cpp 防幻觉通用建议），image 必须在 text 之前。只用 user 消息，不用 system role。

## Output Structure

```
output/
  pages/
    page_001.md
    page_002.md
    ...
```

## Architecture (packages)

```
scansage/
├── main.go                  # CLI entrypoint (urfave/cli.v3)
├── cmd/
│   └── scansage.go          # action handler, pipeline orchestration
├── internal/
│   ├── render/
│   │   └── render.go        # go-fitz PDF → []image.Image
│   ├── ocr/
│   │   └── ocr.go           # llama.cpp API 调用 (image + OCR markdown prompt)
│   └── output/
│       └── output.go        # 写 output/pages/page_NNN.md
```

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| DPI 默认值 | 300 | 专业 OCR 标准，中文小字号可辨识 |
| PNG 落盘 | 纯内存 | 减少 IO，避免临时文件残留 |
| Sidecar 管理 | 用户自行管理 | 简化 scansage 职责 |
| 无配置文件 | — | CLI flags 足够，遵循 KISS |
| 并发 | 默认 1，可选 >1 | 防止多请求压垮 llama.cpp |

## Boundary

- PNG 纯内存，不落盘
- 无配置文件、环境变量
- 用户自行管理 llama.cpp sidecar
- 后续整理（refine + abstract）由用户在工具外自行调度

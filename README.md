# go-imgfeed

[![Go Reference](https://pkg.go.dev/badge/github.com/ultramcu/go-imgfeed.svg)](https://pkg.go.dev/github.com/ultramcu/go-imgfeed)
[![CI](https://github.com/ultramcu/go-imgfeed/actions/workflows/ci.yml/badge.svg)](https://github.com/ultramcu/go-imgfeed/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultramcu/go-imgfeed)](https://goreportcard.com/report/github.com/ultramcu/go-imgfeed)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Load an image from anywhere and **feed it to a multimodal LLM in one call** —
from Go.

Every Go LLM SDK supports image input, but they all leave the same three
chores to you: read the bytes, figure out the MIME type, and assemble the
`data:<mime>;base64,<...>` URL — then wrap it in that SDK's content-part
struct. `go-imgfeed` does all of it, adds optional **downscaling to a token /
byte budget** and an **image token-cost estimate**, and hands the result to
**any** SDK through a thin adapter.

```go
img, _ := imgfeed.FromFile("photo.png",
    imgfeed.WithMaxDim(1024),          // downscale to control cost
    imgfeed.WithDetail(imgfeed.High))

fmt.Println(img.EstimateTokens("gpt-4o")) // budget before you send
url := img.DataURL()                       // ready for any image_url field
```

## Why

- **One call, any source** — `FromFile`, `FromBytes`, `FromReader`, `FromURL`,
  `FromImage` (an `image.Image`).
- **MIME auto-detection** from the actual magic bytes (`http.DetectContentType`),
  with a file-extension fallback — no more guessing `"image/jpeg"`.
- **Budget control** — `WithMaxDim` downscales (aspect-preserving) and
  `WithMaxBytes` re-encodes to fit a size limit, so you don't blow your token
  budget or hit a provider's image-size cap.
- **Cost estimate** — `EstimateTokens(model)` implements OpenAI's tile-based
  image formula (detail-aware), so you can predict input cost up front.
- **Provider-agnostic** — the same loaded image drops into the official OpenAI
  SDK, the community `sashabaranov/go-openai`, `langchaingo`, or the Anthropic
  SDK. Switch providers without re-writing your image plumbing.
- **No bloat** — the core package imports only `golang.org/x/image`. Each SDK
  adapter lives in its own subpackage and imports only its own SDK, so you pull
  in just the one you use.

## Install

```sh
go get github.com/ultramcu/go-imgfeed
```

Requires Go 1.25+ (the floor set by the bundled SDK adapters).

## Core API

```go
img, err := imgfeed.FromFile("cat.png")        // or FromBytes/FromReader/FromURL/FromImage
img.MIME            // "image/png"
img.Width, img.Height
img.DataURL()       // "data:image/png;base64,..."
img.Base64()        // raw base64, no prefix
img.EstimateTokens("gpt-4o")
```

Options: `WithMaxDim(px)`, `WithMaxBytes(n)`, `WithFormat(imgfeed.PNG|imgfeed.JPEG)`,
`WithJPEGQuality(q)`, `WithMIME(m)`, `WithDetail(imgfeed.Auto|Low|High)`,
`WithHTTPClient(c)`.

## Adapters

Each adapter turns an `*imgfeed.Image` into a content part for one SDK.

### OpenAI — official `openai/openai-go`

```go
import (
    "github.com/openai/openai-go/v3"
    "github.com/ultramcu/go-imgfeed"
    "github.com/ultramcu/go-imgfeed/openaidapter"
)

img, _ := imgfeed.FromFile("photo.png", imgfeed.WithDetail(imgfeed.High))
msg := openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
    openaidapter.Text("What is in this image?"),
    openaidapter.Part(img),
})
```

### OpenAI — community `sashabaranov/go-openai`

```go
import (
    openai "github.com/sashabaranov/go-openai"
    "github.com/ultramcu/go-imgfeed"
    "github.com/ultramcu/go-imgfeed/sashadapter"
)

img, _ := imgfeed.FromFile("photo.png")
msg := openai.ChatCompletionMessage{
    Role: openai.ChatMessageRoleUser,
    MultiContent: []openai.ChatMessagePart{
        sashadapter.Text("What is in this image?"),
        sashadapter.Part(img),
    },
}
```

### langchaingo

```go
import (
    "github.com/tmc/langchaingo/llms"
    "github.com/ultramcu/go-imgfeed"
    "github.com/ultramcu/go-imgfeed/lcadapter"
)

img, _ := imgfeed.FromFile("photo.png")
msg := llms.MessageContent{
    Role: llms.ChatMessageTypeHuman,
    Parts: []llms.ContentPart{
        lcadapter.Text("What is in this image?"),
        lcadapter.Part(img), // raw bytes + MIME; langchaingo serializes per provider
    },
}
// lcadapter.URLPart(img) is also available (data-URL form with detail).
```

### Anthropic — `anthropics/anthropic-sdk-go`

```go
import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/ultramcu/go-imgfeed"
    "github.com/ultramcu/go-imgfeed/anthropicadapter"
)

img, _ := imgfeed.FromFile("photo.png")
msg := anthropic.NewUserMessage(
    anthropicadapter.Text("What is in this image?"),
    anthropicadapter.Block(img), // inline base64 image block
)
```

## Notes

- `EstimateTokens` is an approximation: it follows OpenAI's documented tile
  formula and varies by model (the `mini`/`nano` tiers scale up to match
  text-token pricing). Unknown models fall back to the `gpt-4o` cost.
- Anthropic has no per-image "detail" concept, so `WithDetail` is ignored by
  `anthropicadapter`.
- Decoders for PNG, JPEG, GIF, WebP, BMP and TIFF are registered; re-encoding
  (for resizing/byte budgets) outputs PNG or JPEG.

## License

MIT — see [LICENSE](LICENSE).

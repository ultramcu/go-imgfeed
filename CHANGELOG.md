# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-05-24

Initial release.

### Added

- Core `imgfeed` package: load images from bytes, files, readers, URLs or an
  `image.Image` into a provider-agnostic `Image` (bytes + detected MIME +
  dimensions + detail hint).
- MIME auto-detection from magic bytes (`http.DetectContentType`) with a
  file-extension fallback.
- Optional normalization: `WithMaxDim` (downscale, aspect-preserving),
  `WithMaxBytes` (fit a byte budget), `WithFormat` (PNG/JPEG), `WithMIME`,
  `WithJPEGQuality`, `WithDetail`, `WithHTTPClient`.
- `Image.DataURL` / `Image.Base64` output helpers.
- `Image.EstimateTokens` — approximate OpenAI tile-based image token cost,
  detail-aware, with a per-model table.
- SDK adapter subpackages (each importing only its own SDK):
  - `sashadapter` for `github.com/sashabaranov/go-openai`
  - `openaidapter` for `github.com/openai/openai-go/v3`
  - `lcadapter` for `github.com/tmc/langchaingo`
  - `anthropicadapter` for `github.com/anthropics/anthropic-sdk-go`

[0.1.0]: https://github.com/ultramcu/go-imgfeed/releases/tag/v0.1.0

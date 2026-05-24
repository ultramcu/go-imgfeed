package imgfeed

import "net/http"

// Detail mirrors the OpenAI image "detail" hint, which controls how much
// detail the model extracts from an image and therefore how many tokens it
// costs. It is carried on the resulting [Image], applied by the SDK adapters
// that support it, and used by [Image.EstimateTokens].
type Detail string

const (
	// Auto lets the provider decide the detail level. It is the default.
	Auto Detail = "auto"
	// Low requests a low-resolution, fixed-cost reading of the image.
	Low Detail = "low"
	// High requests a high-resolution, tile-based reading of the image.
	High Detail = "high"
)

// Format is an output image encoding used when an image must be re-encoded
// (because of resizing, a byte budget, or an explicit conversion) or when it
// is built from an image.Image. Only the lossless PNG and lossy JPEG
// encoders are supported.
type Format string

const (
	// PNG selects lossless PNG encoding.
	PNG Format = "image/png"
	// JPEG selects lossy JPEG encoding (see [WithJPEGQuality]).
	JPEG Format = "image/jpeg"
)

type config struct {
	detail      Detail
	maxDim      int
	maxBytes    int
	format      Format
	mime        string
	jpegQuality int
	httpClient  *http.Client
}

func defaultConfig() config {
	return config{
		detail:      Auto,
		jpegQuality: 85,
		httpClient:  http.DefaultClient,
	}
}

func buildConfig(opts []Option) config {
	cfg := defaultConfig()
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if cfg.httpClient == nil {
		cfg.httpClient = http.DefaultClient
	}
	if cfg.jpegQuality <= 0 || cfg.jpegQuality > 100 {
		cfg.jpegQuality = 85
	}
	return cfg
}

// Option customizes how an image is loaded and normalized.
type Option func(*config)

// WithDetail sets the OpenAI detail hint (Auto, Low or High). The default is
// Auto. The value is stored on the [Image], forwarded by the adapters that
// support it, and used by [Image.EstimateTokens].
func WithDetail(d Detail) Option { return func(c *config) { c.detail = d } }

// WithMaxDim downscales the image so that neither side exceeds px pixels,
// preserving the aspect ratio. Images already within the bound are left
// untouched. A value <= 0 (the default) disables resizing.
func WithMaxDim(px int) Option { return func(c *config) { c.maxDim = px } }

// WithMaxBytes ensures the encoded image stays at or below n bytes by
// progressively lowering JPEG quality and/or downscaling. It is best effort:
// if the floor is reached the smallest attempt is returned. A value <= 0
// (the default) disables the limit.
func WithMaxBytes(n int) Option { return func(c *config) { c.maxBytes = n } }

// WithFormat forces the output encoding (PNG or JPEG); the image is always
// re-encoded to this format. When unset, original bytes are preserved unless
// a resize or byte budget forces a re-encode, in which case the source format
// is kept where possible (otherwise PNG).
func WithFormat(f Format) Option { return func(c *config) { c.format = f } }

// WithMIME overrides MIME detection, e.g. when the bytes carry a format whose
// signature is not auto-detected.
func WithMIME(mime string) Option { return func(c *config) { c.mime = mime } }

// WithJPEGQuality sets the JPEG quality (1-100) used when encoding to JPEG.
// The default is 85. Out-of-range values are reset to 85.
func WithJPEGQuality(q int) Option { return func(c *config) { c.jpegQuality = q } }

// WithHTTPClient sets the HTTP client used by [FromURL]. It defaults to
// http.DefaultClient.
func WithHTTPClient(h *http.Client) Option { return func(c *config) { c.httpClient = h } }

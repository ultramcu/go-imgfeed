package imgfeed

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	_ "image/gif" // register GIF decoder

	_ "golang.org/x/image/bmp"  // register BMP decoder
	_ "golang.org/x/image/tiff" // register TIFF decoder
	_ "golang.org/x/image/webp" // register WebP decoder
)

// Errors returned by the loaders.
var (
	// ErrEmpty is returned when the source contains no data.
	ErrEmpty = errors.New("imgfeed: empty image data")
	// ErrNotImage is returned when the data is not a recognized image type.
	ErrNotImage = errors.New("imgfeed: data is not a recognized image type")
)

// Image is a normalized, ready-to-send image: the encoded bytes plus the
// detected MIME type, decoded dimensions (0 if unknown) and the chosen
// detail hint. Use [Image.DataURL] for a value to drop into any image_url
// field, or one of the adapter subpackages to build an SDK-specific content
// part.
type Image struct {
	// Data holds the encoded image bytes (possibly re-encoded by resizing).
	Data []byte
	// MIME is the image media type, e.g. "image/png".
	MIME string
	// Width and Height are the pixel dimensions, or 0 if they could not be
	// determined.
	Width, Height int
	// Detail is the resolved detail hint (see [WithDetail]).
	Detail Detail
}

// FromBytes loads an image from raw bytes.
func FromBytes(b []byte, opts ...Option) (*Image, error) {
	return load(b, "", opts)
}

// FromFile loads an image from a file on disk. The file name is used as a
// fallback for MIME detection when the bytes themselves are ambiguous.
func FromFile(path string, opts ...Option) (*Image, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("imgfeed: read file: %w", err)
	}
	return load(b, filepath.Base(path), opts)
}

// FromReader loads an image by reading r to completion.
func FromReader(r io.Reader, opts ...Option) (*Image, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("imgfeed: read: %w", err)
	}
	return load(b, "", opts)
}

// FromURL fetches an image over HTTP(S) and loads it. The request honors ctx
// and the client set by [WithHTTPClient] (default http.DefaultClient).
func FromURL(ctx context.Context, rawURL string, opts ...Option) (*Image, error) {
	cfg := buildConfig(opts)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("imgfeed: request: %w", err)
	}
	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imgfeed: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("imgfeed: fetch %s: unexpected status %s", rawURL, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("imgfeed: read body: %w", err)
	}
	name := ""
	if u, perr := url.Parse(rawURL); perr == nil {
		name = filepath.Base(u.Path)
	}
	return load(b, name, opts)
}

// FromImage encodes an in-memory image.Image. The output format defaults to
// PNG and can be set with [WithFormat]. Resizing and byte-budget options
// apply as usual.
func FromImage(img image.Image, opts ...Option) (*Image, error) {
	if img == nil {
		return nil, ErrEmpty
	}
	cfg := buildConfig(opts)
	f := cfg.format
	if f == "" {
		f = PNG
	}
	b, _, err := encode(img, f, cfg.jpegQuality)
	if err != nil {
		return nil, err
	}
	return load(b, "", opts)
}

func load(b []byte, name string, opts []Option) (*Image, error) {
	if len(b) == 0 {
		return nil, ErrEmpty
	}
	cfg := buildConfig(opts)

	mimeType := cfg.mime
	if mimeType == "" {
		mimeType = detectMIME(b, name)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, fmt.Errorf("%w: detected %q", ErrNotImage, mimeType)
	}

	// Best-effort dimensions from the header alone.
	var w, h int
	if dc, _, err := image.DecodeConfig(bytes.NewReader(b)); err == nil {
		w, h = dc.Width, dc.Height
	}

	needResize := cfg.maxDim > 0 && (w > cfg.maxDim || h > cfg.maxDim)
	needBytes := cfg.maxBytes > 0 && len(b) > cfg.maxBytes
	needFormat := cfg.format != "" && string(cfg.format) != mimeType
	if needResize || needBytes || needFormat {
		nb, nm, nw, nh, err := reencode(b, mimeType, cfg)
		if err != nil {
			return nil, err
		}
		b, mimeType, w, h = nb, nm, nw, nh
	}

	return &Image{Data: b, MIME: mimeType, Width: w, Height: h, Detail: cfg.detail}, nil
}

// detectMIME sniffs the content type from the magic bytes, falling back to
// the file-name extension when the signature is not recognized as an image.
func detectMIME(b []byte, name string) string {
	ct := http.DetectContentType(b)
	if strings.HasPrefix(ct, "image/") {
		return ct
	}
	if name != "" {
		if ext := filepath.Ext(name); ext != "" {
			if t := mime.TypeByExtension(ext); t != "" {
				if i := strings.IndexByte(t, ';'); i >= 0 {
					t = strings.TrimSpace(t[:i])
				}
				return t
			}
		}
	}
	return ct
}

func reencode(b []byte, srcMIME string, cfg config) (out []byte, outMIME string, w, h int, err error) {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("imgfeed: decode for re-encode: %w", err)
	}
	if cfg.maxDim > 0 {
		img = scaleToMax(img, cfg.maxDim)
	}

	f := cfg.format
	if f == "" {
		f = formatForMIME(srcMIME)
	}
	out, outMIME, err = encode(img, f, cfg.jpegQuality)
	if err != nil {
		return nil, "", 0, 0, err
	}
	if cfg.maxBytes > 0 && len(out) > cfg.maxBytes {
		out, outMIME, img, err = shrinkToBytes(img, f, cfg)
		if err != nil {
			return nil, "", 0, 0, err
		}
	}
	bnd := img.Bounds()
	return out, outMIME, bnd.Dx(), bnd.Dy(), nil
}

func scaleToMax(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= maxDim && sh <= maxDim {
		return src
	}
	scale := float64(maxDim) / float64(max(sw, sh))
	nw := max(int(float64(sw)*scale), 1)
	nh := max(int(float64(sh)*scale), 1)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func encode(img image.Image, f Format, q int) ([]byte, string, error) {
	var buf bytes.Buffer
	switch f {
	case JPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", fmt.Errorf("imgfeed: jpeg encode: %w", err)
		}
		return buf.Bytes(), "image/jpeg", nil
	case PNG, "":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("imgfeed: png encode: %w", err)
		}
		return buf.Bytes(), "image/png", nil
	default:
		return nil, "", fmt.Errorf("imgfeed: unsupported output format %q", f)
	}
}

func formatForMIME(m string) Format {
	if m == "image/jpeg" {
		return JPEG
	}
	return PNG
}

// shrinkToBytes downscales (and, for JPEG, lowers quality) until the encoded
// image fits cfg.maxBytes or a floor is reached.
func shrinkToBytes(img image.Image, f Format, cfg config) ([]byte, string, image.Image, error) {
	q := cfg.jpegQuality
	cur := img
	for i := 0; i < 16; i++ {
		out, m, err := encode(cur, f, q)
		if err != nil {
			return nil, "", nil, err
		}
		if len(out) <= cfg.maxBytes {
			return out, m, cur, nil
		}
		if f == JPEG && q > 25 {
			q -= 15
			continue
		}
		b := cur.Bounds()
		nw := int(float64(b.Dx()) * 0.8)
		nh := int(float64(b.Dy()) * 0.8)
		if nw < 16 || nh < 16 {
			return out, m, cur, nil
		}
		cur = scaleToMax(cur, max(nw, nh))
	}
	out, m, err := encode(cur, f, q)
	return out, m, cur, err
}

// Base64 returns the standard base64 encoding of the image bytes.
func (im *Image) Base64() string {
	return base64.StdEncoding.EncodeToString(im.Data)
}

// DataURL returns the image as an RFC 2397 data URL,
// "data:<mime>;base64,<...>", suitable for any image_url field.
func (im *Image) DataURL() string {
	return "data:" + im.MIME + ";base64," + im.Base64()
}

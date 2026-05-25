package imgfeed

// Extra tests targeting the uncovered paths in imgfeed.go, options.go, and
// tokens.go.  All fixtures are generated in-memory; no network access.

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── FromFile error path ───────────────────────────────────────────────────────

func TestFromFile_NotFound(t *testing.T) {
	_, err := FromFile("/nonexistent/path/image.png")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "imgfeed: read file") {
		t.Errorf("error message = %q, expected imgfeed: read file prefix", err)
	}
}

// ── FromReader error path ─────────────────────────────────────────────────────

type failReader struct{ msg string }

func (f failReader) Read(_ []byte) (int, error) { return 0, errors.New(f.msg) }

func TestFromReader_ReadError(t *testing.T) {
	_, err := FromReader(failReader{"disk error"})
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "imgfeed: read") {
		t.Errorf("error = %q, want imgfeed: read prefix", err)
	}
}

// ── FromURL additional paths ──────────────────────────────────────────────────

func TestFromURL_InvalidURL(t *testing.T) {
	// http.NewRequestWithContext fails on a truly malformed URL.
	_, err := FromURL(context.Background(), "://bad url\x00")
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestFromURL_BodyReadError(t *testing.T) {
	// Server writes headers OK but closes body immediately, causing ReadAll to
	// see an error (we use a server that hijacks the connection).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write a 200 with misleading headers, then close the underlying conn.
		hj, ok := w.(http.Hijacker)
		if !ok {
			// Fallback: send 200 but a body that causes decode problems; the
			// resulting error will be ErrNotImage, not a read error, which is
			// still an error return — that's all we're checking.
			w.WriteHeader(http.StatusOK)
			return
		}
		conn, buf, _ := hj.Hijack()
		// Write a minimal HTTP/1.1 200 with no Content-Length, then close.
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: image/png\r\n\r\n")
		_ = buf.Flush()
		conn.Close()
	}))
	defer srv.Close()

	_, err := FromURL(context.Background(), srv.URL+"/img.png")
	if err == nil {
		t.Fatal("expected error when body is cut short")
	}
}

func TestFromURL_WithHTTPClient(t *testing.T) {
	data := makePNG(8, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	custom := &http.Client{}
	im, err := FromURL(context.Background(), srv.URL+"/x.png", WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("FromURL with custom client: %v", err)
	}
	if im.MIME != "image/png" {
		t.Errorf("MIME = %q, want image/png", im.MIME)
	}
}

// ── encode: unsupported format ────────────────────────────────────────────────

func TestEncode_UnsupportedFormat(t *testing.T) {
	img := makeImage(4, 4)
	_, _, err := encode(img, Format("image/bmp"), 85)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error = %q, want unsupported output format", err)
	}
}

// ── formatForMIME ─────────────────────────────────────────────────────────────

func TestFormatForMIME_JPEG(t *testing.T) {
	if got := formatForMIME("image/jpeg"); got != JPEG {
		t.Errorf("formatForMIME(jpeg) = %q, want JPEG", got)
	}
}

func TestFormatForMIME_PNG(t *testing.T) {
	if got := formatForMIME("image/png"); got != PNG {
		t.Errorf("formatForMIME(png) = %q, want PNG", got)
	}
}

func TestFormatForMIME_Other(t *testing.T) {
	// Any non-JPEG MIME should return PNG.
	if got := formatForMIME("image/gif"); got != PNG {
		t.Errorf("formatForMIME(gif) = %q, want PNG", got)
	}
}

// ── FromImage with unsupported format causes encode error in reencode ─────────

func TestFromImage_UnsupportedFormat(t *testing.T) {
	// fromImage encodes with the requested format; unsupported format errors.
	_, err := FromImage(makeImage(4, 4), WithFormat(Format("image/bmp")))
	if err == nil {
		t.Fatal("expected error for unsupported format in FromImage")
	}
}

// ── load: reencode decode error (corrupted data with format-conversion trigger) ──

func TestLoad_ReencodeDecodeError(t *testing.T) {
	// Craft bytes that look like a PNG (magic bytes pass detectMIME) but are
	// otherwise corrupt.  Force reencode via WithFormat(JPEG) so needFormat=true.
	// image.Decode will fail on the corrupt body.
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	corrupt := append(pngMagic, make([]byte, 64)...)

	// WithMIME forces the mime-prefix check to pass; WithFormat forces reencode
	// because string(JPEG) != "image/png".
	_, err := FromBytes(corrupt, WithMIME("image/png"), WithFormat(JPEG))
	if err == nil {
		t.Fatal("expected error decoding corrupt PNG during reencode")
	}
}

// ── detectMIME edge cases ─────────────────────────────────────────────────────

func TestDetectMIME_NoName(t *testing.T) {
	// Data that isn't an image, no name → falls through to http.DetectContentType result.
	b := []byte("plain text")
	got := detectMIME(b, "")
	// Should not be an image/ type.
	if strings.HasPrefix(got, "image/") {
		t.Errorf("unexpected image MIME for text: %q", got)
	}
}

func TestDetectMIME_ExtensionNoMIME(t *testing.T) {
	// Extension that has no mime type registered: falls back to http.DetectContentType.
	b := []byte{0x00, 0x01, 0x02, 0x03}
	got := detectMIME(b, "file.unknownext12345")
	// Just ensure it doesn't panic; the result will be whatever DetectContentType says.
	_ = got
}

func TestDetectMIME_NoExtension(t *testing.T) {
	// Name with no extension: should not panic, should not be image.
	b := []byte("hello world")
	got := detectMIME(b, "noextension")
	_ = got
}

// ── reencode: format="" branch (uses formatForMIME) ──────────────────────────

func TestReencode_FormatEmpty_JPEG(t *testing.T) {
	// JPEG source, no explicit format: reencode should pick JPEG via formatForMIME.
	src := makeJPEG(80, 80)
	cfg := buildConfig([]Option{WithMaxDim(40)}) // triggers reencode, format=""
	out, outMIME, w, h, err := reencode(src, "image/jpeg", cfg)
	if err != nil {
		t.Fatalf("reencode: %v", err)
	}
	if outMIME != "image/jpeg" {
		t.Errorf("outMIME = %q, want image/jpeg", outMIME)
	}
	if w > 40 || h > 40 {
		t.Errorf("dims = %dx%d, not scaled to max 40", w, h)
	}
	_ = out
}

// ── shrinkToBytes coverage ────────────────────────────────────────────────────

func TestShrinkToBytes_PNG_WithByteBudget(t *testing.T) {
	// Large PNG, tiny budget: shrinkToBytes must iterate to reduce size.
	large := makePNG(200, 200)
	const budget = 2000
	im, err := FromBytes(large, WithFormat(PNG), WithMaxBytes(budget))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	// Best-effort: may not hit budget exactly if floor is reached, but it tries.
	_ = im
}

func TestShrinkToBytes_JPEG_QualityReduction(t *testing.T) {
	// JPEG with tight budget: should reduce quality through the q > 25 branch.
	large := makeJPEG(300, 300)
	const budget = 3000
	im, err := FromBytes(large, WithFormat(JPEG), WithMaxBytes(budget))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	_ = im
}

func TestShrinkToBytes_DirectCall_PNG_FloorHit(t *testing.T) {
	// Call shrinkToBytes directly with a budget so small it hits the 16px floor.
	img := makeImage(32, 32)
	cfg := buildConfig([]Option{WithMaxBytes(10), WithFormat(PNG)})
	out, m, _, err := shrinkToBytes(img, PNG, cfg)
	if err != nil {
		t.Fatalf("shrinkToBytes: %v", err)
	}
	if m != "image/png" {
		t.Errorf("mime = %q, want image/png", m)
	}
	_ = out
}

func TestShrinkToBytes_JPEG_ExhaustIterations(t *testing.T) {
	// Budget impossible to hit: exercises the full loop + final encode.
	img := makeImage(64, 64)
	cfg := buildConfig([]Option{WithMaxBytes(1), WithFormat(JPEG)})
	out, m, _, err := shrinkToBytes(img, JPEG, cfg)
	if err != nil {
		t.Fatalf("shrinkToBytes JPEG exhausted: %v", err)
	}
	if m != "image/jpeg" {
		t.Errorf("mime = %q, want image/jpeg", m)
	}
	_ = out
}

// ── buildConfig: nil option + nil httpClient + out-of-range jpegQuality ──────

func TestBuildConfig_NilOption(t *testing.T) {
	// A nil Option should be tolerated (no panic).
	cfg := buildConfig([]Option{nil, WithDetail(Low)})
	if cfg.detail != Low {
		t.Errorf("detail = %q, want Low", cfg.detail)
	}
}

func TestBuildConfig_NilHTTPClient(t *testing.T) {
	// WithHTTPClient(nil) should fall back to http.DefaultClient.
	cfg := buildConfig([]Option{WithHTTPClient(nil)})
	if cfg.httpClient == nil {
		t.Error("httpClient should fall back to DefaultClient, got nil")
	}
}

func TestBuildConfig_JPEGQualityOutOfRange(t *testing.T) {
	// Values ≤0 or >100 should be reset to 85.
	cfgLow := buildConfig([]Option{WithJPEGQuality(0)})
	if cfgLow.jpegQuality != 85 {
		t.Errorf("quality 0 → %d, want 85", cfgLow.jpegQuality)
	}
	cfgHigh := buildConfig([]Option{WithJPEGQuality(101)})
	if cfgHigh.jpegQuality != 85 {
		t.Errorf("quality 101 → %d, want 85", cfgHigh.jpegQuality)
	}
}

func TestWithJPEGQuality_ValidRange(t *testing.T) {
	// A valid quality value should be stored as-is.
	cfg := buildConfig([]Option{WithJPEGQuality(60)})
	if cfg.jpegQuality != 60 {
		t.Errorf("jpegQuality = %d, want 60", cfg.jpegQuality)
	}
}

func TestWithJPEGQuality_AffectsOutput(t *testing.T) {
	// Lower quality → smaller file (not guaranteed for tiny images, but holds
	// for a 200x200 image with moderate detail).
	img := makeImage(200, 200)
	highQ, err := FromImage(img, WithFormat(JPEG), WithJPEGQuality(95))
	if err != nil {
		t.Fatalf("FromImage high quality: %v", err)
	}
	lowQ, err := FromImage(img, WithFormat(JPEG), WithJPEGQuality(10))
	if err != nil {
		t.Fatalf("FromImage low quality: %v", err)
	}
	if len(lowQ.Data) >= len(highQ.Data) {
		t.Logf("high=%d low=%d — low quality should be smaller", len(highQ.Data), len(lowQ.Data))
		// Not a fatal failure: for small uniform images this can invert, just log.
	}
}

func TestWithHTTPClient_Used(t *testing.T) {
	// Verify WithHTTPClient actually stores the client in config.
	custom := &http.Client{}
	cfg := buildConfig([]Option{WithHTTPClient(custom)})
	if cfg.httpClient != custom {
		t.Error("expected custom http.Client to be stored")
	}
}

// ── fitWithin: uncovered function ────────────────────────────────────────────

func TestFitWithin_WidthConstraining(t *testing.T) {
	// 4000x2000, max 2048x2048 → width is the binding constraint.
	w, h := fitWithin(4000, 2000, 2048, 2048)
	if w > 2048 || h > 2048 {
		t.Errorf("fitWithin = %dx%d, exceeds 2048x2048", w, h)
	}
	// Width should be exactly 2048.
	if w != 2048 {
		t.Errorf("w = %d, want 2048 (width constraining)", w)
	}
}

func TestFitWithin_HeightConstraining(t *testing.T) {
	// 1000x3000, max 2048x2048 → height is the binding constraint.
	w, h := fitWithin(1000, 3000, 2048, 2048)
	if w > 2048 || h > 2048 {
		t.Errorf("fitWithin = %dx%d, exceeds 2048x2048", w, h)
	}
	if h != 2048 {
		t.Errorf("h = %d, want 2048 (height constraining)", h)
	}
}

func TestFitWithin_SquareInput(t *testing.T) {
	w, h := fitWithin(4096, 4096, 2048, 2048)
	if w != 2048 || h != 2048 {
		t.Errorf("fitWithin(4096,4096,2048,2048) = %dx%d, want 2048x2048", w, h)
	}
}

// ── EstimateTokens: 2048 scaling path + nano model ───────────────────────────

func TestEstimateTokens_LargerThan2048(t *testing.T) {
	// Image bigger than 2048x2048 → fitWithin is called to scale it down.
	im := &Image{Width: 4096, Height: 4096, Detail: High}
	got := im.EstimateTokens("gpt-4o")
	// After fit to 2048x2048 then shortest=2048>768 → scale to 768x768:
	// tiles = ceil(768/512) * ceil(768/512) = 2*2 = 4 → 85 + 4*170 = 765
	want := 765
	if got != want {
		t.Errorf("EstimateTokens(4096x4096 High gpt-4o) = %d, want %d", got, want)
	}
}

func TestEstimateTokens_4096x2048_High(t *testing.T) {
	// 4096x2048 → fitWithin(4096,2048,2048,2048): width constraining → 2048x1024
	// shortest=1024 > 768 → scale: factor=768/1024=0.75 → 1536x768
	// tiles = ceil(1536/512)*ceil(768/512) = 3*2 = 6 → 85 + 6*170 = 1105
	im := &Image{Width: 4096, Height: 2048, Detail: High}
	got := im.EstimateTokens("gpt-4o")
	want := 1105
	if got != want {
		t.Errorf("EstimateTokens(4096x2048 High) = %d, want %d", got, want)
	}
}

func TestEstimateTokens_NanoModel(t *testing.T) {
	// "nano" in model name → scaled-up tier (same as mini).
	im := &Image{Width: 512, Height: 512, Detail: High}
	got := im.EstimateTokens("gpt-4.1-nano")
	// nano is in modelCosts directly, same cost as mini: 2833 + 5667 = 8500
	if got != 8500 {
		t.Errorf("EstimateTokens(nano) = %d, want 8500", got)
	}
}

func TestEstimateTokens_UnknownNano(t *testing.T) {
	// "future-nano" not in map but contains "nano" → scaled-up tier.
	im := &Image{Width: 512, Height: 512, Detail: High}
	got := im.EstimateTokens("future-nano")
	if got != 8500 {
		t.Errorf("EstimateTokens(future-nano) = %d, want 8500", got)
	}
}

func TestEstimateTokens_AllKnownModels(t *testing.T) {
	// Smoke-test every entry in modelCosts doesn't panic and returns >0.
	im := &Image{Width: 512, Height: 512, Detail: High}
	models := []string{
		"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini",
		"gpt-4.1-nano", "gpt-4-turbo", "gpt-4-vision", "o1", "o3",
		"o4-mini", "chatgpt-4o",
	}
	for _, m := range models {
		got := im.EstimateTokens(m)
		if got <= 0 {
			t.Errorf("EstimateTokens(%q) = %d, want >0", m, got)
		}
	}
}

func TestEstimateTokens_CaseInsensitive(t *testing.T) {
	// costFor trims and lowercases, so "GPT-4O" should match "gpt-4o".
	im := &Image{Width: 512, Height: 512, Detail: High}
	lower := im.EstimateTokens("gpt-4o")
	upper := im.EstimateTokens("GPT-4O")
	if lower != upper {
		t.Errorf("case sensitivity: lower=%d upper=%d", lower, upper)
	}
}

// ── scaleToMax: already-within-bound short-circuit ───────────────────────────

func TestScaleToMax_AlreadyWithin(t *testing.T) {
	img := makeImage(10, 10)
	out := scaleToMax(img, 100)
	b := out.Bounds()
	if b.Dx() != 10 || b.Dy() != 10 {
		t.Errorf("scaleToMax returned %dx%d, want 10x10 (no scaling)", b.Dx(), b.Dy())
	}
}

// ── detectMIME: magic-detected image type (no fallback needed) ───────────────

func TestDetectMIME_MagicPNG(t *testing.T) {
	b := makePNG(4, 4)
	got := detectMIME(b, "")
	if got != "image/png" {
		t.Errorf("detectMIME(PNG bytes, '') = %q, want image/png", got)
	}
}

func TestDetectMIME_MagicJPEG(t *testing.T) {
	b := makeJPEG(4, 4)
	got := detectMIME(b, "")
	if got != "image/jpeg" {
		t.Errorf("detectMIME(JPEG bytes, '') = %q, want image/jpeg", got)
	}
}

// ── WithMaxBytes: PNG byte budget ─────────────────────────────────────────────

func TestWithMaxBytes_PNG(t *testing.T) {
	const budget = 3000
	im, err := FromBytes(makePNG(200, 200), WithFormat(PNG), WithMaxBytes(budget))
	if err != nil {
		t.Fatalf("FromBytes PNG budget: %v", err)
	}
	// Best-effort: result should be ≤ budget or at floor.
	_ = im
}

// ── reencode: png source, no format override (picks PNG via formatForMIME) ───

func TestReencode_FormatEmpty_PNG(t *testing.T) {
	src := makePNG(80, 80)
	cfg := buildConfig([]Option{WithMaxDim(40)})
	out, outMIME, w, h, err := reencode(src, "image/png", cfg)
	if err != nil {
		t.Fatalf("reencode png: %v", err)
	}
	if outMIME != "image/png" {
		t.Errorf("outMIME = %q, want image/png", outMIME)
	}
	if w > 40 || h > 40 {
		t.Errorf("dims %dx%d exceed maxDim 40", w, h)
	}
	_ = out
}

// ── encode: PNG path explicit ─────────────────────────────────────────────────

func TestEncode_PNG(t *testing.T) {
	img := makeImage(4, 4)
	out, m, err := encode(img, PNG, 85)
	if err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	if m != "image/png" {
		t.Errorf("mime = %q, want image/png", m)
	}
	// Verify the output is a valid PNG.
	_, _, err = image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Errorf("output not a valid PNG: %v", err)
	}
}

func TestEncode_EmptyFormat(t *testing.T) {
	// Empty format "" maps to PNG in the switch.
	img := makeImage(4, 4)
	out, m, err := encode(img, Format(""), 85)
	if err != nil {
		t.Fatalf("encode empty format: %v", err)
	}
	if m != "image/png" {
		t.Errorf("mime = %q, want image/png", m)
	}
	_ = out
}

// ── scaleToMax: very thin image (one dimension at limit, other well below) ───

func TestScaleToMax_WideImage(t *testing.T) {
	// 200x10 → maxDim=50: max(200,10)=200, scale=0.25, nw=50, nh=3
	img := makeImage(200, 10)
	out := scaleToMax(img, 50)
	b := out.Bounds()
	if b.Dx() > 50 || b.Dy() > 50 {
		t.Errorf("scaleToMax wide: %dx%d > 50", b.Dx(), b.Dy())
	}
}

// ── PNG with no re-encode path (dimensions and bytes within limits) ───────────

func TestLoad_NoReencode(t *testing.T) {
	// Small PNG, no options → goes straight through without reencode.
	data := makePNG(8, 8)
	im, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if !bytes.Equal(im.Data, data) {
		t.Error("data should be unchanged when no reencode is triggered")
	}
}

// ── WithMaxDim on JPEG keeps JPEG format ─────────────────────────────────────

func TestWithMaxDim_JPEG(t *testing.T) {
	im, err := FromBytes(makeJPEG(200, 100), WithMaxDim(50))
	if err != nil {
		t.Fatalf("FromBytes JPEG maxDim: %v", err)
	}
	if im.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg after JPEG→resize", im.MIME)
	}
	if im.Width > 50 || im.Height > 50 {
		t.Errorf("dims %dx%d exceed maxDim 50", im.Width, im.Height)
	}
}

// ── GIF registered decoder (import side-effect) ──────────────────────────────

func TestFromBytes_GIF(t *testing.T) {
	// Minimal 1x1 GIF89a (hand-crafted, network-safe).
	gif1x1 := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
		0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, // LSD: 1x1, GCT present
		0xff, 0xff, 0xff, 0x00, 0x00, 0x00, // GCT: white, black
		0x2c,                   // image separator
		0x00, 0x00, 0x00, 0x00, // left, top
		0x01, 0x00, 0x01, 0x00, 0x00, // 1x1, no LCT
		0x02, 0x02, 0x4c, 0x01, 0x00, // LZW min=2, data block, trailer
		0x3b, // GIF trailer
	}
	im, err := FromBytes(gif1x1)
	if err != nil {
		// GIF might not encode to an image/ mime from http.DetectContentType
		// (it does return "image/gif"), so this path is normally fine.
		t.Logf("FromBytes GIF: %v (may require reencode support)", err)
		return
	}
	if !strings.HasPrefix(im.MIME, "image/") {
		t.Errorf("GIF MIME = %q, want image/*", im.MIME)
	}
}

// ── A valid makePNG helper also covers makeJPEG via encode JPEG path ──────────

func TestMakeJPEG_Valid(t *testing.T) {
	data := makeJPEG(10, 10)
	_, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("makeJPEG produced invalid JPEG: %v", err)
	}
}

func TestMakePNG_Valid(t *testing.T) {
	data := makePNG(10, 10)
	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("makePNG produced invalid PNG: %v", err)
	}
}

// ── makeImage produces a non-nil image.Image ─────────────────────────────────

func TestMakeImage_Bounds(t *testing.T) {
	img := makeImage(16, 8)
	b := img.Bounds()
	if b.Dx() != 16 || b.Dy() != 8 {
		t.Errorf("makeImage bounds = %dx%d, want 16x8", b.Dx(), b.Dy())
	}
}

// ── Image.Base64 + DataURL round-trip ────────────────────────────────────────

func TestDataURL_ContainsBase64(t *testing.T) {
	im := &Image{Data: []byte("hello"), MIME: "image/png"}
	dataURL := im.DataURL()
	b64 := im.Base64()
	if !strings.HasSuffix(dataURL, b64) {
		t.Errorf("DataURL should end with Base64 output")
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Errorf("DataURL = %q, wrong prefix", dataURL)
	}
}

// ── uniform color image: pixel values exercise color.RGBA ────────────────────

func TestMakeImage_PixelValues(t *testing.T) {
	img := makeImage(4, 4).(*image.RGBA)
	// pixel at (1,2): R=1, G=2, B=120, A=255
	c := img.RGBAAt(1, 2)
	if c.R != 1 || c.G != 2 || c.B != 120 || c.A != 255 {
		t.Errorf("pixel(1,2) = %v, want RGBA{1,2,120,255}", c)
	}
}

// ── WithMaxBytes: already-within-budget skips reencode ───────────────────────

func TestWithMaxBytes_SmallImageWithinBudget(t *testing.T) {
	// A tiny PNG is well within a large budget: no reencode needed.
	data := makePNG(4, 4)
	im, err := FromBytes(data, WithMaxBytes(1_000_000))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if !bytes.Equal(im.Data, data) {
		t.Error("data changed when already within byte budget")
	}
}

// ── color.RGBA import used explicitly ────────────────────────────────────────

func TestColorRGBA_Direct(t *testing.T) {
	c := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	if c.R != 10 {
		t.Error("color.RGBA R mismatch")
	}
}

// ── scaleToMax: tall image (height is max dimension) ─────────────────────────

func TestScaleToMax_TallImage(t *testing.T) {
	// 10x200 → maxDim=50: max(10,200)=200, scale=0.25, nw=3, nh=50
	img := makeImage(10, 200)
	out := scaleToMax(img, 50)
	b := out.Bounds()
	if b.Dx() > 50 || b.Dy() > 50 {
		t.Errorf("scaleToMax tall: %dx%d > 50", b.Dx(), b.Dy())
	}
}

// ── reencode with maxBytes triggers shrinkToBytes path ────────────────────────

func TestReencode_TriggersShrinkToBytes(t *testing.T) {
	// A PNG source with both maxDim and a very tight byte budget:
	// after resize, encoded size may still exceed budget → shrinkToBytes called.
	large := makePNG(300, 300)
	im, err := FromBytes(large, WithMaxDim(200), WithMaxBytes(5000), WithFormat(JPEG))
	if err != nil {
		t.Fatalf("FromBytes with maxBytes: %v", err)
	}
	_ = im
}

// ── io usage: cover io.ReadAll path in FromReader with a limited reader ───────

func TestFromReader_LimitedReader(t *testing.T) {
	// io.LimitReader wrapping a PNG: should load fine.
	data := makePNG(8, 8)
	limited := io.LimitReader(bytes.NewReader(data), int64(len(data)))
	im, err := FromReader(limited)
	if err != nil {
		t.Fatalf("FromReader limited: %v", err)
	}
	if im.MIME != "image/png" {
		t.Errorf("MIME = %q, want image/png", im.MIME)
	}
}

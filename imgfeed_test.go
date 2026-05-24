package imgfeed

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func makePNG(w, h int) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, makeImage(w, h)); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func makeJPEG(w, h int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, makeImage(w, h), &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestFromBytes_PNG(t *testing.T) {
	im, err := FromBytes(makePNG(40, 20))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.MIME != "image/png" {
		t.Errorf("MIME = %q, want image/png", im.MIME)
	}
	if im.Width != 40 || im.Height != 20 {
		t.Errorf("dims = %dx%d, want 40x20", im.Width, im.Height)
	}
	if im.Detail != Auto {
		t.Errorf("Detail = %q, want auto", im.Detail)
	}
	if !strings.HasPrefix(im.DataURL(), "data:image/png;base64,") {
		t.Errorf("DataURL prefix wrong: %q", im.DataURL()[:32])
	}
	if im.Base64() == "" {
		t.Error("Base64 empty")
	}
}

func TestFromBytes_JPEG(t *testing.T) {
	im, err := FromBytes(makeJPEG(30, 30))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg", im.MIME)
	}
}

func TestFromBytes_NotImage(t *testing.T) {
	_, err := FromBytes([]byte("this is plainly not an image, just some text bytes"))
	if !errors.Is(err, ErrNotImage) {
		t.Errorf("err = %v, want ErrNotImage", err)
	}
}

func TestFromBytes_Empty(t *testing.T) {
	if _, err := FromBytes(nil); !errors.Is(err, ErrEmpty) {
		t.Errorf("err = %v, want ErrEmpty", err)
	}
}

func TestFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(p, makePNG(16, 16), 0o600); err != nil {
		t.Fatal(err)
	}
	im, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if im.MIME != "image/png" || im.Width != 16 {
		t.Errorf("got %q %dx%d", im.MIME, im.Width, im.Height)
	}
}

func TestFromReader(t *testing.T) {
	im, err := FromReader(bytes.NewReader(makePNG(8, 8)))
	if err != nil {
		t.Fatalf("FromReader: %v", err)
	}
	if im.Width != 8 {
		t.Errorf("width = %d, want 8", im.Width)
	}
}

func TestFromURL(t *testing.T) {
	data := makePNG(24, 12)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	im, err := FromURL(context.Background(), srv.URL+"/photo.png")
	if err != nil {
		t.Fatalf("FromURL: %v", err)
	}
	if im.MIME != "image/png" || im.Width != 24 || im.Height != 12 {
		t.Errorf("got %q %dx%d", im.MIME, im.Width, im.Height)
	}
}

func TestFromURL_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := FromURL(context.Background(), srv.URL); err == nil {
		t.Error("expected error on 404")
	}
}

func TestWithMaxDim(t *testing.T) {
	im, err := FromBytes(makePNG(100, 40), WithMaxDim(50))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.Width > 50 || im.Height > 50 {
		t.Errorf("not downscaled: %dx%d", im.Width, im.Height)
	}
	if im.Width != 50 || im.Height != 20 {
		t.Errorf("aspect/scale wrong: got %dx%d, want 50x20", im.Width, im.Height)
	}
	if im.MIME != "image/png" {
		t.Errorf("re-encode kept format wrong: %q", im.MIME)
	}
}

func TestWithMaxDim_NoUpscale(t *testing.T) {
	orig := makePNG(20, 20)
	im, err := FromBytes(orig, WithMaxDim(500))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.Width != 20 || im.Height != 20 {
		t.Errorf("should not upscale: %dx%d", im.Width, im.Height)
	}
	if !bytes.Equal(im.Data, orig) {
		t.Error("bytes should be untouched when within bound")
	}
}

func TestWithFormat_JPEG(t *testing.T) {
	im, err := FromBytes(makePNG(32, 32), WithFormat(JPEG))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg", im.MIME)
	}
	if !strings.HasPrefix(im.DataURL(), "data:image/jpeg;base64,") {
		t.Errorf("DataURL = %q...", im.DataURL()[:32])
	}
}

func TestWithMIME(t *testing.T) {
	im, err := FromBytes(makePNG(8, 8), WithMIME("image/png"))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.MIME != "image/png" {
		t.Errorf("MIME = %q", im.MIME)
	}
}

func TestWithDetail(t *testing.T) {
	im, err := FromBytes(makePNG(8, 8), WithDetail(High))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if im.Detail != High {
		t.Errorf("Detail = %q, want high", im.Detail)
	}
}

func TestFromImage_DefaultPNG(t *testing.T) {
	im, err := FromImage(makeImage(20, 10))
	if err != nil {
		t.Fatalf("FromImage: %v", err)
	}
	if im.MIME != "image/png" {
		t.Errorf("MIME = %q, want image/png", im.MIME)
	}
	if im.Width != 20 || im.Height != 10 {
		t.Errorf("dims = %dx%d, want 20x10", im.Width, im.Height)
	}
}

func TestFromImage_JPEG(t *testing.T) {
	im, err := FromImage(makeImage(20, 10), WithFormat(JPEG))
	if err != nil {
		t.Fatalf("FromImage: %v", err)
	}
	if im.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg", im.MIME)
	}
}

func TestFromImage_Nil(t *testing.T) {
	if _, err := FromImage(nil); !errors.Is(err, ErrEmpty) {
		t.Errorf("err = %v, want ErrEmpty", err)
	}
}

func TestWithMaxBytes(t *testing.T) {
	const budget = 4000
	im, err := FromBytes(makePNG(300, 300), WithFormat(JPEG), WithMaxBytes(budget))
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if len(im.Data) > budget {
		t.Errorf("len = %d, want <= %d", len(im.Data), budget)
	}
}

func TestDetectMIME_ExtensionFallback(t *testing.T) {
	// Bytes that DetectContentType cannot classify as an image; the .png
	// extension should be used as the fallback.
	got := detectMIME([]byte{0x00, 0x01, 0x02, 0x03}, "photo.png")
	if got != "image/png" {
		t.Errorf("detectMIME = %q, want image/png", got)
	}
}

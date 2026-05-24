package anthropicadapter

import (
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	imgfeed "github.com/ultramcu/go-imgfeed"
)

// tinyPNG is a minimal 1x1 PNG. The exact bytes are unimportant for these
// tests; only that they round-trip through base64.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89,
}

func TestBlock(t *testing.T) {
	im := &imgfeed.Image{
		Data:   tinyPNG,
		MIME:   "image/png",
		Detail: imgfeed.High, // must be ignored by Block
	}

	got := Block(im)

	if got.OfImage == nil {
		t.Fatal("Block did not produce an image block (OfImage is nil)")
	}
	src := got.OfImage.Source.OfBase64
	if src == nil {
		t.Fatal("image block has no base64 source (Source.OfBase64 is nil)")
	}

	if want := im.Base64(); src.Data != want {
		t.Errorf("base64 data = %q, want %q", src.Data, want)
	}
	if want := anthropic.Base64ImageSourceMediaTypeImagePNG; src.MediaType != want {
		t.Errorf("media type = %q, want %q", src.MediaType, want)
	}
	// The raw base64 must not carry a data URL prefix.
	if len(src.Data) >= 5 && src.Data[:5] == "data:" {
		t.Errorf("base64 data unexpectedly carries a data URL prefix: %q", src.Data)
	}
}

func TestBlockArbitraryMIME(t *testing.T) {
	// im.MIME is passed through verbatim; the SDK takes a plain string.
	im := &imgfeed.Image{Data: []byte{0x01, 0x02, 0x03}, MIME: "image/webp"}

	got := Block(im)

	src := got.OfImage.Source.OfBase64
	if src == nil {
		t.Fatal("image block has no base64 source")
	}
	if want := anthropic.Base64ImageSourceMediaTypeImageWebP; src.MediaType != want {
		t.Errorf("media type = %q, want %q", src.MediaType, want)
	}
	if want := im.Base64(); src.Data != want {
		t.Errorf("base64 data = %q, want %q", src.Data, want)
	}
}

func TestText(t *testing.T) {
	got := Text("hi")

	if got.OfText == nil {
		t.Fatal("Text did not produce a text block (OfText is nil)")
	}
	if got.OfText.Text != "hi" {
		t.Errorf("text = %q, want %q", got.OfText.Text, "hi")
	}
	if got.OfImage != nil {
		t.Errorf("Text unexpectedly set an image block: %+v", got.OfImage)
	}
}

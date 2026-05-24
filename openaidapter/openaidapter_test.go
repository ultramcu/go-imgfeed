package openaidapter

import (
	"testing"

	imgfeed "github.com/ultramcu/go-imgfeed"
)

func TestPart(t *testing.T) {
	im := &imgfeed.Image{
		Data:   []byte{0x89, 0x50, 0x4e, 0x47}, // tiny PNG signature bytes
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	part := Part(im)

	if part.OfImageURL == nil {
		t.Fatalf("Part did not produce an image_url content part: %+v", part)
	}

	gotURL := part.OfImageURL.ImageURL.URL
	if want := im.DataURL(); gotURL != want {
		t.Errorf("image URL = %q, want %q", gotURL, want)
	}

	if got := part.OfImageURL.ImageURL.Detail; got != string(imgfeed.High) {
		t.Errorf("detail = %q, want %q", got, string(imgfeed.High))
	}
}

func TestText(t *testing.T) {
	part := Text("hi")

	if part.OfText == nil {
		t.Fatalf("Text did not produce a text content part: %+v", part)
	}

	if got := part.OfText.Text; got != "hi" {
		t.Errorf("text = %q, want %q", got, "hi")
	}
}

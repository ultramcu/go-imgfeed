package lcadapter

import (
	"bytes"
	"testing"

	imgfeed "github.com/ultramcu/go-imgfeed"

	"github.com/tmc/langchaingo/llms"
)

func TestPart(t *testing.T) {
	im := &imgfeed.Image{
		Data:   []byte{1, 2, 3},
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	got := Part(im)

	bc, ok := got.(llms.BinaryContent)
	if !ok {
		t.Fatalf("Part returned %T, want llms.BinaryContent", got)
	}
	if bc.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want %q", bc.MIMEType, "image/png")
	}
	if !bytes.Equal(bc.Data, im.Data) {
		t.Errorf("Data = %v, want %v", bc.Data, im.Data)
	}
}

func TestURLPart(t *testing.T) {
	im := &imgfeed.Image{
		Data:   []byte{1, 2, 3},
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	got := URLPart(im)

	iuc, ok := got.(llms.ImageURLContent)
	if !ok {
		t.Fatalf("URLPart returned %T, want llms.ImageURLContent", got)
	}
	if want := im.DataURL(); iuc.URL != want {
		t.Errorf("URL = %q, want %q", iuc.URL, want)
	}
	if iuc.Detail != string(imgfeed.High) {
		t.Errorf("Detail = %q, want %q", iuc.Detail, string(imgfeed.High))
	}
}

func TestText(t *testing.T) {
	got := Text("hi")

	tc, ok := got.(llms.TextContent)
	if !ok {
		t.Fatalf("Text returned %T, want llms.TextContent", got)
	}
	if tc.Text != "hi" {
		t.Errorf("Text = %q, want %q", tc.Text, "hi")
	}
}

package genaidapter

import (
	"bytes"
	"testing"

	imgfeed "github.com/ultramcu/go-imgfeed"
)

func TestPart(t *testing.T) {
	im := &imgfeed.Image{Data: []byte{1, 2, 3}, MIME: "image/png", Detail: imgfeed.High}
	p := Part(im)
	if p.InlineData == nil {
		t.Fatal("Part: InlineData is nil")
	}
	if p.InlineData.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want image/png", p.InlineData.MIMEType)
	}
	if !bytes.Equal(p.InlineData.Data, im.Data) {
		t.Errorf("Data = %v, want %v", p.InlineData.Data, im.Data)
	}
	if p.Text != "" {
		t.Errorf("Text = %q, want empty", p.Text)
	}
}

func TestText(t *testing.T) {
	p := Text("hi")
	if p.Text != "hi" {
		t.Errorf("Text = %q, want hi", p.Text)
	}
	if p.InlineData != nil {
		t.Error("Text: InlineData should be nil")
	}
}

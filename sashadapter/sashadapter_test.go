package sashadapter

import (
	"testing"

	openai "github.com/sashabaranov/go-openai"
	imgfeed "github.com/ultramcu/go-imgfeed"
)

func TestPart(t *testing.T) {
	im := &imgfeed.Image{
		Data:   []byte{0x89, 0x50, 0x4e, 0x47}, // "\x89PNG" header bytes
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	got := Part(im)

	if got.Type != openai.ChatMessagePartTypeImageURL {
		t.Errorf("Type = %q, want %q", got.Type, openai.ChatMessagePartTypeImageURL)
	}
	if got.ImageURL == nil {
		t.Fatal("ImageURL is nil, want non-nil")
	}
	if want := im.DataURL(); got.ImageURL.URL != want {
		t.Errorf("ImageURL.URL = %q, want %q", got.ImageURL.URL, want)
	}
	if got.ImageURL.Detail != openai.ImageURLDetailHigh {
		t.Errorf("ImageURL.Detail = %q, want %q", got.ImageURL.Detail, openai.ImageURLDetailHigh)
	}
}

func TestPartDetailMapping(t *testing.T) {
	cases := []struct {
		name string
		in   imgfeed.Detail
		want openai.ImageURLDetail
	}{
		{"auto", imgfeed.Auto, openai.ImageURLDetailAuto},
		{"low", imgfeed.Low, openai.ImageURLDetailLow},
		{"high", imgfeed.High, openai.ImageURLDetailHigh},
		{"empty falls back to auto", imgfeed.Detail(""), openai.ImageURLDetailAuto},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			im := &imgfeed.Image{Data: []byte{0x01}, MIME: "image/png", Detail: tc.in}
			got := Part(im)
			if got.ImageURL == nil {
				t.Fatal("ImageURL is nil, want non-nil")
			}
			if got.ImageURL.Detail != tc.want {
				t.Errorf("Detail = %q, want %q", got.ImageURL.Detail, tc.want)
			}
		})
	}
}

func TestText(t *testing.T) {
	got := Text("hi")

	if got.Type != openai.ChatMessagePartTypeText {
		t.Errorf("Type = %q, want %q", got.Type, openai.ChatMessagePartTypeText)
	}
	if got.Text != "hi" {
		t.Errorf("Text = %q, want %q", got.Text, "hi")
	}
	if got.ImageURL != nil {
		t.Errorf("ImageURL = %+v, want nil", got.ImageURL)
	}
}

// Package genaidapter turns an [imgfeed.Image] into a content part for the
// Google Gen AI SDK (google.golang.org/genai), used by Gemini models.
//
// The returned parts go into a [genai.Content] (typically via
// [genai.NewContentFromParts] with [genai.RoleUser]), which is passed to
// Models.GenerateContent:
//
//	img, _ := imgfeed.FromFile("photo.png")
//	content := genai.NewContentFromParts([]*genai.Part{
//		genaidapter.Text("What is in this image?"),
//		genaidapter.Part(img),
//	}, genai.RoleUser)
//
// The image is embedded inline as a base64 Blob (MIME type + raw bytes). The
// Gemini API has no per-image "detail" concept, so [imgfeed.WithDetail] is
// ignored here. For images larger than the inline request limit, upload via
// the SDK's File API instead.
package genaidapter

import (
	imgfeed "github.com/ultramcu/go-imgfeed"
	"google.golang.org/genai"
)

// Part returns an inline image content part carrying the image's bytes and
// MIME type.
func Part(im *imgfeed.Image) *genai.Part {
	return genai.NewPartFromBytes(im.Data, im.MIME)
}

// Text returns a text content part.
func Text(s string) *genai.Part {
	return genai.NewPartFromText(s)
}

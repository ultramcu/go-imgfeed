// Package lcadapter converts an [imgfeed.Image] into content parts for the
// LLM framework github.com/tmc/langchaingo.
//
// The returned [llms.ContentPart] values are meant to populate the Parts
// field of an [llms.MessageContent], which is then passed to a model's
// GenerateContent method, for example:
//
//	msg := llms.MessageContent{
//		Role: llms.ChatMessageTypeHuman,
//		Parts: []llms.ContentPart{
//			lcadapter.Text("what is this?"),
//			lcadapter.Part(img),
//		},
//	}
//	resp, err := model.GenerateContent(ctx, []llms.MessageContent{msg})
//
// Prefer [Part], which carries the raw bytes plus MIME type as a binary
// part and lets langchaingo serialize the image in the form each provider
// expects. Use [URLPart] when you specifically want an image-URL part backed
// by the image's data URL, and [Text] for a plain text part.
package lcadapter

import (
	imgfeed "github.com/ultramcu/go-imgfeed"

	"github.com/tmc/langchaingo/llms"
)

// Part returns the preferred image content part: a binary part carrying the
// image's raw bytes and MIME type. langchaingo serializes a binary part into
// whatever each provider's API expects, so this works across backends without
// the caller building a data URL.
func Part(im *imgfeed.Image) llms.ContentPart {
	return llms.BinaryPart(im.MIME, im.Data)
}

// URLPart returns an image-URL content part backed by the image's data URL
// (im.DataURL()). The image's detail hint is passed through to the SDK's
// detail field. Use this when a provider/path expects an image_url rather
// than inline binary data; otherwise prefer [Part].
func URLPart(im *imgfeed.Image) llms.ContentPart {
	return llms.ImageURLWithDetailPart(im.DataURL(), string(im.Detail))
}

// Text returns a plain text content part, provided for symmetry with [Part]
// and [URLPart] when assembling multi-part messages.
func Text(s string) llms.ContentPart {
	return llms.TextPart(s)
}

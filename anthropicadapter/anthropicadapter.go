// Package anthropicadapter converts an [imgfeed.Image] into content blocks for
// the official SDK github.com/anthropics/anthropic-sdk-go.
//
// Unlike the OpenAI-style SDKs, Anthropic does not accept a data URL. An image
// is sent as a base64 image source block carrying a media type (e.g.
// "image/png") and the raw base64 data (no "data:" prefix). Anthropic also has
// no per-image "detail" concept, so im.Detail is ignored.
//
// The returned [anthropic.ContentBlockParamUnion] values are meant to populate
// the content of a message, typically via [anthropic.NewUserMessage] (or by
// appending to an [anthropic.MessageParam].Content), for example:
//
//	msg := anthropic.NewUserMessage(
//		anthropicadapter.Text("what is this?"),
//		anthropicadapter.Block(img),
//	)
//
// Use [Block] for a base64 image block backed by the image bytes and [Text] for
// a plain text block.
package anthropicadapter

import (
	anthropic "github.com/anthropics/anthropic-sdk-go"
	imgfeed "github.com/ultramcu/go-imgfeed"
)

// Block returns a base64 image content block for use in an Anthropic message's
// content (see [anthropic.NewUserMessage]). The image is embedded inline as a
// base64 source built from im.MIME (the media type) and im.Base64() (the raw
// base64 data, without a "data:" prefix). im.Detail is ignored because the
// Anthropic API has no per-image detail setting.
//
// The SDK's [anthropic.NewImageBlockBase64] accepts the media type as a plain
// string and converts it to the typed [anthropic.Base64ImageSourceMediaType]
// internally, so im.MIME is passed through verbatim.
func Block(im *imgfeed.Image) anthropic.ContentBlockParamUnion {
	return anthropic.NewImageBlockBase64(im.MIME, im.Base64())
}

// Text returns a plain text content block, provided for symmetry with [Block]
// when assembling multi-block messages.
func Text(s string) anthropic.ContentBlockParamUnion {
	return anthropic.NewTextBlock(s)
}

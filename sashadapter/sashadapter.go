// Package sashadapter converts an [imgfeed.Image] into content parts for the
// community SDK github.com/sashabaranov/go-openai.
//
// The returned [openai.ChatMessagePart] values are meant to populate the
// MultiContent field of an [openai.ChatCompletionMessage], for example:
//
//	msg := openai.ChatCompletionMessage{
//		Role: openai.ChatMessageRoleUser,
//		MultiContent: []openai.ChatMessagePart{
//			sashadapter.Text("what is this?"),
//			sashadapter.Part(img),
//		},
//	}
//
// Use [Part] for an image (image_url) part backed by the image's data URL and
// [Text] for a plain text part.
package sashadapter

import (
	openai "github.com/sashabaranov/go-openai"
	imgfeed "github.com/ultramcu/go-imgfeed"
)

// Part returns an image content part (type image_url) for use in
// [openai.ChatCompletionMessage].MultiContent. The image is embedded inline via
// im.DataURL(), and im.Detail is mapped to the SDK's [openai.ImageURLDetail].
func Part(im *imgfeed.Image) openai.ChatMessagePart {
	return openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeImageURL,
		ImageURL: &openai.ChatMessageImageURL{
			URL:    im.DataURL(),
			Detail: detail(im.Detail),
		},
	}
}

// Text returns a text content part (type text) for use in
// [openai.ChatCompletionMessage].MultiContent, provided for symmetry with
// [Part] when assembling multi-part messages.
func Text(s string) openai.ChatMessagePart {
	return openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: s,
	}
}

// detail maps an imgfeed detail hint to the SDK's ImageURLDetail. Unknown
// values fall back to auto, matching the SDK's default behavior.
func detail(d imgfeed.Detail) openai.ImageURLDetail {
	switch d {
	case imgfeed.Low:
		return openai.ImageURLDetailLow
	case imgfeed.High:
		return openai.ImageURLDetailHigh
	default:
		return openai.ImageURLDetailAuto
	}
}

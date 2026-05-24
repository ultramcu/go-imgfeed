// Package openaidapter adapts an [imgfeed.Image] into content parts for the
// official OpenAI Go SDK (github.com/openai/openai-go/v3), targeting the Chat
// Completions multimodal message format.
//
// A user message's content can be either a plain string or an array of content
// parts. To send an image you build the array form and pass it to
// [openai.UserMessage]:
//
//	img, _ := imgfeed.FromFile("cat.png")
//	msg := openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
//		openaidapter.Text("What is in this image?"),
//		openaidapter.Part(img),
//	})
//	// msg is a ChatCompletionMessageParamUnion ready for params.Messages.
//
// [Part] produces an image_url content part whose URL is a self-contained data
// URL ([imgfeed.Image.DataURL]), so no separate upload or hosting is required,
// and whose detail level is mapped from the image's [imgfeed.Detail] hint.
// [Text] produces a plain text content part.
package openaidapter

import (
	"github.com/openai/openai-go/v3"
	imgfeed "github.com/ultramcu/go-imgfeed"
)

// Part returns an image_url content part for use in a user message's content
// array. The image is embedded as a base64 data URL via [imgfeed.Image.DataURL],
// and the detail hint is carried through from im.Detail.
//
// In openai-go v3.37.0 the image_url "detail" field is a plain string accepting
// "auto", "low" or "high"; imgfeed.Detail uses the same string values, so the
// mapping is a direct conversion.
func Part(im *imgfeed.Image) openai.ChatCompletionContentPartUnionParam {
	return openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
		URL:    im.DataURL(),
		Detail: string(im.Detail),
	})
}

// Text returns a text content part carrying s, for use in a user message's
// content array alongside one or more image parts from [Part].
func Text(s string) openai.ChatCompletionContentPartUnionParam {
	return openai.TextContentPart(s)
}

// Package imgfeed loads images from files, byte slices, readers, URLs or
// image.Image values and normalizes them into a small, provider-agnostic
// [Image] that is ready to "feed" to a multimodal LLM.
//
// It removes the three menial steps every Go LLM SDK leaves to the caller:
// reading the bytes, detecting the MIME type, and assembling the
// "data:<mime>;base64,<...>" URL. On top of that it can optionally downscale
// an image to a pixel or byte budget (see [WithMaxDim] and [WithMaxBytes])
// and estimate the number of input tokens it will cost (see
// [Image.EstimateTokens]).
//
// The core package has no LLM SDK dependencies. To turn an [Image] into a
// content part for a specific SDK, import one of the adapter subpackages:
//
//   - github.com/ultramcu/go-imgfeed/sashadapter    (sashabaranov/go-openai)
//   - github.com/ultramcu/go-imgfeed/openaidapter   (openai/openai-go)
//   - github.com/ultramcu/go-imgfeed/lcadapter      (tmc/langchaingo)
//   - github.com/ultramcu/go-imgfeed/anthropicadapter (anthropics/anthropic-sdk-go)
//
// Each adapter imports only its own SDK, so importing the core (or one
// adapter) never pulls in the others.
//
// Basic usage:
//
//	img, err := imgfeed.FromFile("photo.png",
//		imgfeed.WithMaxDim(1024),
//		imgfeed.WithDetail(imgfeed.High))
//	if err != nil {
//		// handle error
//	}
//	url := img.DataURL()                 // ready for any image_url field
//	cost := img.EstimateTokens("gpt-4o") // approximate input tokens
package imgfeed

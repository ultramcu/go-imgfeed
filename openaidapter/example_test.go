package openaidapter_test

import (
	"fmt"

	"github.com/openai/openai-go/v3"
	imgfeed "github.com/ultramcu/go-imgfeed"
	"github.com/ultramcu/go-imgfeed/openaidapter"
)

// Example shows assembling a multimodal user message: a text part plus an image
// part, passed to openai.UserMessage. This builds the request value only; no
// network call is made.
func Example() {
	img := &imgfeed.Image{
		Data:   []byte{0x89, 0x50, 0x4e, 0x47}, // tiny PNG signature bytes
		MIME:   "image/png",
		Detail: imgfeed.Auto,
	}

	content := []openai.ChatCompletionContentPartUnionParam{
		openaidapter.Text("describe"),
		openaidapter.Part(img),
	}

	msg := openai.UserMessage(content)

	// The message is a user message with a 2-element content array.
	parts := msg.OfUser.Content.OfArrayOfContentParts
	fmt.Println(len(parts))
	fmt.Println(parts[0].OfText.Text)
	// Output:
	// 2
	// describe
}

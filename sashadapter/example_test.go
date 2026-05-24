package sashadapter_test

import (
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	imgfeed "github.com/ultramcu/go-imgfeed"
	"github.com/ultramcu/go-imgfeed/sashadapter"
)

// Example shows how to assemble a multi-part user message for the
// github.com/sashabaranov/go-openai SDK, combining a text prompt with an
// inline image part.
func Example() {
	// In real code, load the image with imgfeed.FromBytes / FromFile / FromURL.
	// Here we construct one directly so the example is deterministic.
	img := &imgfeed.Image{
		Data:   []byte{0x89, 0x50, 0x4e, 0x47},
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	msg := openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser,
		MultiContent: []openai.ChatMessagePart{
			sashadapter.Text("what is this?"),
			sashadapter.Part(img),
		},
	}

	fmt.Println(msg.Role)
	fmt.Println(len(msg.MultiContent))
	fmt.Println(msg.MultiContent[0].Type)
	fmt.Println(msg.MultiContent[1].Type)
	fmt.Println(msg.MultiContent[1].ImageURL.Detail)
	// Output:
	// user
	// 2
	// text
	// image_url
	// high
}

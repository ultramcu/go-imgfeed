package lcadapter_test

import (
	imgfeed "github.com/ultramcu/go-imgfeed"
	"github.com/ultramcu/go-imgfeed/lcadapter"

	"github.com/tmc/langchaingo/llms"
)

// Example assembles a multimodal human message from a text part and an image
// part. The resulting [llms.MessageContent] is what you pass to a model's
// GenerateContent method.
func Example() {
	img := &imgfeed.Image{
		Data:   []byte{0x89, 0x50, 0x4e, 0x47}, // "\x89PNG" header bytes
		MIME:   "image/png",
		Detail: imgfeed.High,
	}

	msg := llms.MessageContent{
		Role: llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{
			lcadapter.Text("what is this?"),
			lcadapter.Part(img),
		},
	}

	// msg is ready to hand to model.GenerateContent(ctx, []llms.MessageContent{msg}).
	_ = msg
	// Output:
}

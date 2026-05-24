package anthropicadapter_test

import (
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	imgfeed "github.com/ultramcu/go-imgfeed"
	"github.com/ultramcu/go-imgfeed/anthropicadapter"
)

// Example shows assembling an Anthropic user message that mixes a text block and
// an image block built from an [imgfeed.Image]. No network call is made.
func Example() {
	img := &imgfeed.Image{
		Data: []byte{0x89, 0x50, 0x4e, 0x47}, // stand-in image bytes
		MIME: "image/png",
	}

	msg := anthropic.NewUserMessage(
		anthropicadapter.Text("what is this?"),
		anthropicadapter.Block(img),
	)

	fmt.Println(msg.Role)
	fmt.Println(len(msg.Content))
	// Output:
	// user
	// 2
}

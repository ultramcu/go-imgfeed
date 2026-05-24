package genaidapter_test

import (
	"fmt"

	imgfeed "github.com/ultramcu/go-imgfeed"
	"github.com/ultramcu/go-imgfeed/genaidapter"
	"google.golang.org/genai"
)

func Example() {
	// In real code: img, err := imgfeed.FromFile("photo.png")
	img := &imgfeed.Image{MIME: "image/png", Data: []byte("...png bytes...")}

	content := genai.NewContentFromParts([]*genai.Part{
		genaidapter.Text("What is in this image?"),
		genaidapter.Part(img),
	}, genai.RoleUser)

	fmt.Println(content.Role)
	// Output: user
}

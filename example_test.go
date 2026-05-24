package imgfeed_test

import (
	"context"
	"fmt"

	"github.com/ultramcu/go-imgfeed"
)

// Load an image from disk, downscale it to fit a token budget, and inspect
// the result. The DataURL is ready to drop into any provider's image_url
// field; the adapter subpackages turn the Image into an SDK content part.
func ExampleFromFile() {
	img, err := imgfeed.FromFile("photo.png",
		imgfeed.WithMaxDim(1024),
		imgfeed.WithDetail(imgfeed.High))
	if err != nil {
		// handle error
		return
	}

	_ = img.DataURL() // "data:image/png;base64,..." for any image_url field
	fmt.Println(img.MIME, img.EstimateTokens("gpt-4o"))
}

// Fetch a remote image and estimate what it will cost as input.
func ExampleFromURL() {
	img, err := imgfeed.FromURL(context.Background(),
		"https://example.com/cat.jpg",
		imgfeed.WithMaxDim(768))
	if err != nil {
		return
	}
	fmt.Printf("%dx%d ~%d tokens\n", img.Width, img.Height, img.EstimateTokens("gpt-4o"))
}

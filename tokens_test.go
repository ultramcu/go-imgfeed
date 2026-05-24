package imgfeed

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		img   Image
		model string
		want  int
	}{
		// low detail is a flat base cost regardless of size
		{"low gpt-4o", Image{Width: 4000, Height: 4000, Detail: Low}, "gpt-4o", 85},
		// 512x512 high: shortest (512) <= 768, 1 tile -> 85 + 170
		{"high small", Image{Width: 512, Height: 512, Detail: High}, "gpt-4o", 255},
		// auto behaves like high
		{"auto small", Image{Width: 512, Height: 512, Detail: Auto}, "gpt-4o", 255},
		// 1024x1024 high: shortest 1024 -> scaled to 768, 2x2 tiles -> 85 + 4*170
		{"high large", Image{Width: 1024, Height: 1024, Detail: High}, "gpt-4o", 765},
		// unknown dims -> base
		{"unknown dims", Image{Width: 0, Height: 0, Detail: High}, "gpt-4o", 85},
		// mini tier scales up: 512x512 high -> 2833 + 5667
		{"mini small", Image{Width: 512, Height: 512, Detail: High}, "gpt-4o-mini", 8500},
		// unknown model falls back to gpt-4o cost
		{"unknown model", Image{Width: 512, Height: 512, Detail: High}, "totally-made-up", 255},
		// name heuristic: anything with "mini" uses the scaled tier
		{"future mini", Image{Width: 512, Height: 512, Detail: High}, "gpt-9-mini", 8500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.img.EstimateTokens(tt.model); got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

package imgfeed

import "strings"

type modelCost struct {
	base    int
	perTile int
}

// Per-model image token costs for the OpenAI tile formula. The "mini"/"nano"
// tiers report image usage scaled up to match text-token pricing, hence the
// much larger constants. Values are best-effort approximations.
var modelCosts = map[string]modelCost{
	"gpt-4o":       {base: 85, perTile: 170},
	"gpt-4o-mini":  {base: 2833, perTile: 5667},
	"gpt-4.1":      {base: 85, perTile: 170},
	"gpt-4.1-mini": {base: 2833, perTile: 5667},
	"gpt-4.1-nano": {base: 2833, perTile: 5667},
	"gpt-4-turbo":  {base: 85, perTile: 170},
	"gpt-4-vision": {base: 85, perTile: 170},
	"o1":           {base: 85, perTile: 170},
	"o3":           {base: 85, perTile: 170},
	"o4-mini":      {base: 2833, perTile: 5667},
	"chatgpt-4o":   {base: 85, perTile: 170},
}

// EstimateTokens returns an approximate number of input tokens the image will
// cost for the given model, using OpenAI's tile-based image formula:
//
//   - Low detail costs a flat base amount.
//   - High/Auto detail scales the image to fit a 2048x2048 box, then so its
//     shortest side is 768px, and charges base + perTile per 512px tile.
//
// It is an estimate; actual usage may differ slightly and varies by model.
// Unknown models fall back to the gpt-4o cost, and models whose name contains
// "mini" or "nano" use the scaled-up tier. If the image dimensions are
// unknown, the base cost is returned.
func (im *Image) EstimateTokens(model string) int {
	mc := costFor(model)
	if im.Detail == Low {
		return mc.base
	}
	w, h := im.Width, im.Height
	if w <= 0 || h <= 0 {
		return mc.base
	}

	// 1. Scale down to fit within a 2048x2048 square.
	if w > 2048 || h > 2048 {
		w, h = fitWithin(w, h, 2048, 2048)
	}
	// 2. Scale down so the shortest side is 768px.
	shortest := min(w, h)
	if shortest > 768 {
		scale := 768.0 / float64(shortest)
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
	}
	// 3. Count 512px tiles.
	tiles := ((w + 511) / 512) * ((h + 511) / 512)
	if tiles < 1 {
		tiles = 1
	}
	return mc.base + mc.perTile*tiles
}

func costFor(model string) modelCost {
	m := strings.ToLower(strings.TrimSpace(model))
	if mc, ok := modelCosts[m]; ok {
		return mc
	}
	if strings.Contains(m, "mini") || strings.Contains(m, "nano") {
		return modelCost{base: 2833, perTile: 5667}
	}
	return modelCost{base: 85, perTile: 170}
}

func fitWithin(w, h, mw, mh int) (int, int) {
	scale := float64(mw) / float64(w)
	if s := float64(mh) / float64(h); s < scale {
		scale = s
	}
	return int(float64(w) * scale), int(float64(h) * scale)
}

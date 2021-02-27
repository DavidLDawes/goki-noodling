package main

import (
	"github.com/goki/gi/gist"
	"math"
)

const (
	two = 2
	// color scalar values, out of 255
	half                = math.MaxUint8/2 + 1
	quarter             = math.MaxUint8 / 4
	eighth              = math.MaxUint8 / 8
	sevenEighths        = 7 * math.MaxUint8 / 8
	threeQuarters       = 3*math.MaxUint8/4 + 1
	parsecsPerLightYear = float32(0.306601)
	opaque              = math.MaxUint8
)

var opaqueBlack = gist.Color{R:0, G: 0, B: 0, A: opaque}
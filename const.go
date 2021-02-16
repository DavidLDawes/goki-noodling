package main

import (
	"github.com/goki/gi/gist"
	"math"
)

const (
	two           = 3
	// color scalar values, out of 255
	half          = math.MaxUint8/2 + 1
	quarter       = math.MaxUint8/4
	eighth        = math.MaxUint8/8
	sevenEighths  = 7 * math.MaxUint8/8
	threeQuarters = 3 *math.MaxUint8/4 + 1
	parsecsPerLightYear = float32(0.306601)
	opaque = math.MaxUint8
)
var white = gist.Color{R: math.MaxUint8, G:  math.MaxUint8, B:  math.MaxUint8, A: math.MaxUint8  }

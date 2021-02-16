package main

import (
	"encoding/binary"
	"github.com/chewxy/math32"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gist"
	"github.com/spaolacci/murmur3"
	"image/color"
	"math"
	"math/rand"
)

type classDetails struct {
	class       string
	brightColor color.RGBA
	medColor    color.RGBA
	dimColor    color.RGBA
	odds        float32
	fudge       float32
	minMass     float32
	deltaMass   float32
	minRadii    float32
	deltaRadii  float32
	minLum      float32
	deltaLum    float32
	pixels      int32
}

type star struct {
	class       string
	brightColor color.RGBA
	medColor    color.RGBA
	dimColor    color.RGBA
	pixels      int32
	mass        float32
	radii       float32
	luminance   float32
	// 3D position
	x float32
	y float32
	z float32
	// sector location, 0 <= sx, sy, sz < 100
	sx float32
	sy float32
	sz float32
	// display location, 0 <= dx, dy, dz < 1000
	dx float32
	dy float32
	dz float32
}

type sector struct {
	x uint32
	y uint32
	z uint32
}

type position struct {
	x float32
	y float32
	z float32
}

type width struct {
	float32
}

type simpleLine struct {
	from		position
	to 			position
	color		gist.Color
}

var routeColors = []gist.Color{
	gist.Color(color.RGBA{R: math.MaxUint8, G: 0, B: 0, A: math.MaxUint8}),
	gist.Color(color.RGBA{R: math.MaxUint8, G: half + eighth, B: 0, A: math.MaxUint8}),
	gist.Color(color.RGBA{R: math.MaxUint8, G: math.MaxUint8, B: 9, A: math.MaxUint8 - 32}),
	gist.Color(color.RGBA{R: 0, G: 0, B: threeQuarters + eighth, A: math.MaxUint8 - 64}),
}

var lWidth = width{float32: 0.0005}

var (
	bright = uint8(math.MaxUint8)
	tween  = uint8(sevenEighths)
	med    = uint8(threeQuarters)
	dim    = uint8(half)

	classO = classDetails{
		class:       "O",
		brightColor: color.RGBA{R: 0, G: 0, B: bright, A: opaque},
		medColor:    color.RGBA{R: 0, G: 0, B: bright, A: opaque},
		dimColor:    color.RGBA{R: 0, G: 0, B: med, A: opaque},
		odds:        .0000003,
		fudge:       .0000000402,
		minMass:     16.00001,
		deltaMass:   243.2,
		minRadii:    6,
		deltaRadii:  17.3,
		minLum:      30000,
		deltaLum:    147000.2,
		pixels:      11,
	}

	classB = classDetails{
		class:       "B",
		brightColor: color.RGBA{R: dim, G: dim, B: bright, A: opaque},
		medColor:    color.RGBA{R: dim / two, G: dim / two, B: med, A: opaque},
		dimColor: color.RGBA{
			R: dim / (two * two),
			G: dim / (two * two), B: dim, A: opaque,
		},
		odds:       .0013,
		fudge:      .0003,
		minMass:    2.1,
		deltaMass:  13.9,
		minRadii:   1.8,
		deltaRadii: 4.8,
		minLum:     25,
		deltaLum:   29975,
		pixels:     8,
	}

	classA = classDetails{
		class:       "A",
		brightColor: color.RGBA{R: bright, G: bright, B: bright, A: opaque},
		medColor:    color.RGBA{R: med, G: med, B: med, A: opaque},
		dimColor:    color.RGBA{R: dim, G: dim, B: dim, A: opaque},
		odds:        .006,
		fudge:       .0018,
		minMass:     1.4,
		deltaMass:   .7,
		minRadii:    1.4,
		deltaRadii:  .4,
		minLum:      5,
		deltaLum:    20,
		pixels:      6,
	}

	classF = classDetails{
		class:       "F",
		brightColor: color.RGBA{R: bright, G: bright, B: tween, A: opaque},
		medColor:    color.RGBA{R: tween, G: tween, B: dim, A: opaque},
		dimColor:    color.RGBA{R: med, G: med, B: dim / two, A: opaque},
		odds:        .03,
		fudge:       .012,
		minMass:     1.04,
		deltaMass:   .36,
		minRadii:    1.15,
		deltaRadii:  .25,
		minLum:      1.5,
		deltaLum:    3.5,
		pixels:      5,
	}

	classG = classDetails{
		class:       "G",
		brightColor: color.RGBA{R: tween, G: tween, B: 0, A: opaque},
		medColor:    color.RGBA{R: med, G: med, B: 0, A: opaque},
		dimColor:    color.RGBA{R: dim, G: dim, B: 0, A: opaque},
		odds:        .076,
		fudge:       .01102,
		minMass:     .8,
		deltaMass:   .24,
		minRadii:    .96,
		deltaRadii:  .19,
		minLum:      .6,
		deltaLum:    .9,
		pixels:      4,
	}

	classK = classDetails{
		class:       "K",
		brightColor: color.RGBA{R: 0xFE, G: 0xD8, B: 0xB1, A: opaque},
		medColor:    color.RGBA{R: 3 * (0xFE / 4), G: 3 * (0xD8 / 4), B: 3 * (0xB1 / 4), A: opaque},
		dimColor:    color.RGBA{R: 0xFE / two, G: uint8(0xD8) / two, B: uint8(0xB1) / two, A: opaque},
		odds:        .121,
		fudge:       .042,
		minMass:     .45,
		deltaMass:   .35,
		minRadii:    .7,
		deltaRadii:  .26,
		minLum:      .08,
		deltaLum:    .52,
		pixels:      3,
	}

	classM = classDetails{
		class:       "M",
		brightColor: color.RGBA{R: bright, G: 0, B: 0, A: opaque},
		medColor:    color.RGBA{R: med, G: 0, B: 0, A: opaque},
		dimColor:    color.RGBA{R: dim, G: 0, B: 0, A: opaque},
		odds:        .7645,
		fudge:       .04,
		minMass:     1.04,
		deltaMass:   .36,
		minRadii:    1.15,
		deltaRadii:  .25,
		minLum:      1.5,
		deltaLum:    3.5,
		pixels:      2,
	}

	starDetailsByClass = [7]classDetails{classO, classB, classA, classF, classG, classK, classM}
	classByZoom        = [11]int{7, 7, 7, 7, 7, 7, 6, 5, 4, 3, 2}
)

func getStarDetails(classDetails classDetails, sector sector, random1m *rand.Rand) []star {
	stars := make([]star, 0)
	loopSize := int32(423.728813559 * (classDetails.odds - classDetails.fudge + 2*classDetails.fudge*random1m.Float32()))
	for i := 0; i < int(loopSize); i++ {
		nextStar := star{}
		random1 := random1m.Float32()
		nextStar.sx = random1m.Float32()
		nextStar.sy = random1m.Float32()
		nextStar.sz = random1m.Float32()
		nextStar.x = float32(sector.x) + nextStar.sx
		nextStar.y = float32(sector.y) + nextStar.sy
		nextStar.z = float32(sector.z) + nextStar.sz
		nextStar.class = classDetails.class
		nextStar.brightColor = classDetails.brightColor
		nextStar.dimColor = classDetails.dimColor
		nextStar.mass = classDetails.minMass + classDetails.deltaMass*(1+random1)
		nextStar.radii = classDetails.minRadii + random1*classDetails.deltaRadii
		nextStar.luminance = classDetails.minLum + random1*classDetails.deltaLum
		nextStar.pixels = classDetails.pixels
		stars = append(stars, nextStar)
	}

	return stars
}

func getSectorDetails(fromSector sector) []star {
	result := make([]star, 0)
	random1m := getHash(fromSector)
	classCount := 0
	for _, starDetails := range starDetailsByClass {
		nextClass := getStarDetails(starDetails, fromSector, random1m)
		result = append(result, nextClass...)
		classCount++
		//if classCount > classByZoom[zoomIndex] {
		//	break
		//}
	}

	return result
}

func getSectorFromPosition(now position) sector {
	return sector{uint32(now.x / 100), uint32(now.y / 100), uint32(now.z / 100)}
}

func getHash(aSector sector) *rand.Rand {
	id := murmur3.New64()
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, aSector.x)
	_, err := id.Write(buf)
	if err != nil {
		print("Failed to hash part 1")
	}

	binary.LittleEndian.PutUint32(buf, aSector.y)
	_, err = id.Write(buf)
	if err != nil {
		print("Failed to hash part two")
	}

	binary.LittleEndian.PutUint32(buf, aSector.z)
	_, err = id.Write(buf)
	if err != nil {
		print("Failed to hash part 3")
	}

	return rand.New(rand.NewSource(int64(id.Sum64())))
}

func distance(s1 star, s2 star) float32 {
	return math32.Sqrt((s1.x-s2.x)*(s1.x-s2.x) + (s1.y-s2.y)*(s1.y-s2.y) + (s1.z-s2.z)*(s1.z-s2.z))
}

/*
func getLine(sc *gi3d.Scene, s1 star, s2 star, lColor gist.Color) *gi3d.Solid {
	//lName := "L" + strconv.Itoa(count)
	lnsm := gi3d.AddNewLines(sc, "lines",
			[]mat32.Vec3{ {X: scaled(s1.x), Y: scaled(s1.y), Z: scaled(s1.z)},
			{X: scaled(s2.x), Y: scaled(s2.y), Z: scaled(s2.z) + 9.0}},
			mat32.Vec2{.2, .1},
			gi3d.OpenLines)
	lns := gi3d.AddNewSolid(sc, sc, "lines", lnsm.Name())
	lns.Pose.Pos.Set(0, 0, 10)
	lns.Mat.Color.SetUInt8(255, 255, 0, 128) // alpha = .5
	// sc.Wireframe = true                      // debugging
	lns.Pose.Pos.Set(0, 0, 0)
	return lns
}

func drawLine(sc *gi3d.Scene, id int, lnsm *gi3d.Lines) {
	lin := gi3d.AddNewSolid(sc, sc, "l"+strconv.Itoa(id), lnsm.Name())
	lin.Pose.Pos.Set(0, 0, 1)
	//lns.Mat.Color.SetUInt8(255, 255, 0, 128)
	lin.Mat.Color = white
	lin.Mat.Color.A = math.MaxUint8
}

*/
func checkForRoutes(sc *gi3d.Scene, stars []star, star star, id int) (result []*simpleLine) {
	result = make([]*simpleLine, 0)
	for _, innerStar := range stars[id+1:] {
		routeColor := checkFor1Route(sc, star, innerStar)
		if routeColor.A > 0 {
			result = append(result, &simpleLine{
				from: position{x: star.x, y: star.y, z: star.z},
				to: position{x: innerStar.x, y: innerStar.y, z: innerStar.z},
				color: routeColor})
		}
	}
	return
}

func checkFor1Route(sc *gi3d.Scene, s1 star, s2 star) (result gist.Color) {
	delta := int(distance(s1, s2)*100 * parsecsPerLightYear)
	result = gist.Color{R: 0, G: 0, B: 0, A: 0}
	if delta < len(routeColors) {
		result = routeColors[delta]
	}
	// Return transparent black if there isn't one

	return
}

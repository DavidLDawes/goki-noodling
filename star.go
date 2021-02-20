package main

import (
	"encoding/binary"
	"image/color"
	"math"
	"math/rand"
	"strconv"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gist"
	"github.com/goki/mat32"
	"github.com/spaolacci/murmur3"
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
	id          int
	class       string
	brightColor color.RGBA
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
	routes []*jump
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

type jump struct {
	color    gist.Color
	parsecs  int
	distance float32
	s1ID     int
	s2ID     int
}

type simpleLine struct {
	from  position
	to    position
	route *jump
}

var routeColors = []gist.Color{
	gist.Color(color.RGBA{R: math.MaxUint8, G: 0, B: 0, A: math.MaxUint8}),
	gist.Color(color.RGBA{R: math.MaxUint8, G: half + eighth, B: 0, A: math.MaxUint8}),
	gist.Color(color.RGBA{R: math.MaxUint8, G: math.MaxUint8, B: 9, A: math.MaxUint8 - 8}),
	gist.Color(color.RGBA{R: 0, G: math.MaxUint8, B: 0, A: math.MaxUint8 - 16}),
	gist.Color(color.RGBA{R: quarter, G: quarter, B: math.MaxUint8, A: math.MaxUint8 - 24}),
	gist.Color(color.RGBA{R: math.MaxUint8, G: 0, B: math.MaxUint8, A: math.MaxUint8 - 32}),
}

var (
	bright = uint8(math.MaxUint8)
	tween  = uint8(sevenEighths)
	med    = uint8(threeQuarters)
	dim    = uint8(half)
	noJump = jump{gist.Color(color.RGBA{R: 0, G: 0, B: 0, A: 0}), 500000000.0, 500000000.0, -1, -1}
	noLine = simpleLine{from: position{x: 0, y: 0, z: 0,}, to: position{x: 0, y: 0, z: 0,}, route: &noJump}

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

func getStarDetails(classDetails classDetails, sector sector, random1m *rand.Rand) []*star {
	stars := make([]*star, 0)
	loopSize := int32(1200 * (classDetails.odds - classDetails.fudge + 2*classDetails.fudge*random1m.Float32()))
	for i := 0; i < int(loopSize); i++ {
		nextStar := star{}
		nextStar.id = len(stars)
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
		nextStar.radii = (classDetails.minRadii + random1*classDetails.deltaRadii)/2
		nextStar.luminance = classDetails.minLum + random1*classDetails.deltaLum
		nextStar.pixels = classDetails.pixels
		stars = append(stars, &nextStar)
	}

	return stars
}

func getSectorDetails(fromSector sector) (result []*star) {
	result = make([]*star, 0)
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

func distance(s1 *star, s2 *star) float32 {
	return math32.Sqrt((s1.x-s2.x)*(s1.x-s2.x) + (s1.y-s2.y)*(s1.y-s2.y) + (s1.z-s2.z)*(s1.z-s2.z))
}

func renderStars(sc *gi3d.Scene) {
	stars := make([]*star, 0)
	for x := uint32(0); x < 4; x++ {
		for y := uint32(0); y < 2; y++ {
			sector := sector{x: x, y: y, z: 0}
			for _, star := range getSectorDetails(sector) {
				stars = append(stars, star)
			}
		}
	}
	if len(stars) > 0 {
		lines := make([]*simpleLine, 0)
		sName := "sphere"
		sphm := gi3d.AddNewSphere(sc, sName, 0.002, 24)
		for id, star := range stars {
			sph := gi3d.AddNewSolid(sc, sc, sName, sphm.Name())
			sph.Pose.Pos.Set(star.x-2.5, star.y-1.0, star.z+8.0)
			sph.Mat.Color.SetUInt8(star.brightColor.R, star.brightColor.G, star.brightColor.B, star.brightColor.A)
			for _, route := range checkForRoutes(sc, stars, star, id) {
				lines = append(lines, route)
			}
		}

		for id, lin := range lines {
			thickness := float32(0.001)
			if lin.route.color.A < math.MaxUint8-23 {
				thickness = 0.0001
			} else if lin.route.color.A < math.MaxUint8-15 {
				thickness = 0.0002
			} else if lin.route.color.A < math.MaxUint8-7 {
				thickness = 0.0003
			}
			lnsm := gi3d.AddNewLines(sc, "Lines-"+strconv.Itoa(id),
				[]mat32.Vec3{
					{X: lin.from.x - 2.5, Y: lin.from.y - 1.0, Z: lin.from.z + 8.0},
					{X: lin.to.x - 2.5, Y: lin.to.y - 1.0, Z: lin.to.z + 8.0},
				},
				mat32.Vec2{X: thickness, Y: thickness},
				gi3d.OpenLines,
			)
			solidLine := gi3d.AddNewSolid(sc, sc, "Lines-"+strconv.Itoa(id), lnsm.Name())
			// solidLine.Pose.Pos.Set(lin.from.x - .5, lin.from.y - .5, lin.from.z + 8)
			// lns.Mat.Color.SetUInt8(255, 255, 0, 128)
			solidLine.Mat.Color = lin.route.color
		}

	}
}

func checkForRoutes(sc *gi3d.Scene, stars []*star, star *star, id int) (result []*simpleLine) {
	tempResult := make([]*simpleLine, 0)
	result = make([]*simpleLine, 0)
	for innerId, innerStar := range stars {
		if innerId == id {
			continue
		}
		routeColor := checkFor1Route(sc, star, innerStar)
		if routeColor.color.A > 0 {
			routeColor.s1ID = star.id
			routeColor.s2ID = id
			newRoute := &simpleLine{
				from:  position{x: star.x, y: star.y, z: star.z},
				to:    position{x: innerStar.x, y: innerStar.y, z: innerStar.z},
				route: routeColor,
			}
			tempResult = append(tempResult, newRoute)
		}
	}
	closest := []*simpleLine{&noLine, &noLine,}
	star.routes = make([]*jump, 0)
	if len(tempResult) > 2 {
		for _, nextSimpleLine := range tempResult {
			if nextSimpleLine.route.distance < closest[0].route.distance {
				if closest[0].route.distance < closest[1].route.distance {
					closest[1] = closest[0]
				}
				closest[0] = nextSimpleLine
			} else if nextSimpleLine.route.distance < tempResult[1].route.distance {
				closest[1] = nextSimpleLine
			}
		}
		tempResult = closest
	}
	for _, nextSimpleLine := range tempResult {
		star.routes = append(star.routes, nextSimpleLine.route)
	}
	result = append(result, tempResult...)
	return
}

func checkFor1Route(sc *gi3d.Scene, s1 *star, s2 *star) (result *jump) {
	routeLength := distance(s1, s2) * 100 * parsecsPerLightYear
	delta := int(routeLength)
	if delta < len(routeColors) {
		result = &jump{routeColors[delta], delta, routeLength, s1.id, s2.id}
		s1.routes = append(s1.routes, result)
	} else {
		result = &noJump
	}
	// Return transparent black if there isn't one

	return
}

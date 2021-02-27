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
	from     position
	to       position
	jumpInfo *jump
}

const (
	intensityStep = 8
)

var (
	offsets = position{x: -2.5, y: -1.0, z: 0.0}

	intensity = []uint8{
		0, 0, intensityStep, 2 * intensityStep, 3 * intensityStep, 4 * intensityStep, 2 * intensityStep,
	}
	jumpColors = []gist.Color{
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: 0, B: 0, A: math.MaxUint8 - intensity[0]}),
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: half + eighth - eighth, B: 0, A: math.MaxUint8 - intensity[1]}),
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: math.MaxUint8 - eighth, B: 0, A: math.MaxUint8 - intensity[2]}),
		gist.Color(color.RGBA{R: 0, G: math.MaxUint8 - eighth, B: 0, A: math.MaxUint8 - intensity[3]}),
		gist.Color(color.RGBA{R: 0, G: 0, B: math.MaxUint8 - eighth, A: math.MaxUint8 - intensity[4]}),
		// gist.Color(color.RGBA{R: math.MaxUint8, G: 0, B: math.MaxUint8, A: math.MaxUint8 - intensity[step++]}),//
	}

	tween  = uint8(sevenEighths)
	med    = uint8(threeQuarters)
	dim    = uint8(half)
	noJump = jump{gist.Color(color.RGBA{R: 0, G: 0, B: 0, A: 0}), 500000000.0, 500000000.0, -1, -1}
	noLine = simpleLine{from: position{x: 0, y: 0, z: 0}, to: position{x: 0, y: 0, z: 0}, jumpInfo: &noJump}

	jumpsByStar = make(map[int][]*jump)

	classO = classDetails{
		class:       "O",
		brightColor: color.RGBA{R: 0, G: 0, B: tween, A: opaque},
		medColor:    color.RGBA{R: 0, G: 0, B: tween, A: opaque},
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
		brightColor: color.RGBA{R: dim, G: dim, B: tween, A: opaque},
		medColor:    color.RGBA{R: dim / two, G: dim / two, B: half, A: opaque},
		dimColor:    color.RGBA{R: dim / (two * two), G: dim / (two * two), B: tween / two, A: opaque},
		odds:        .0013,
		fudge:       .0003,
		minMass:     2.1,
		deltaMass:   13.9,
		minRadii:    1.8,
		deltaRadii:  4.8,
		minLum:      25,
		deltaLum:    29975,
		pixels:      8,
	}

	classA = classDetails{
		class:       "A",
		brightColor: color.RGBA{R: tween, G: tween, B: tween, A: opaque},
		medColor:    color.RGBA{R: sevenEighths, G: sevenEighths, B: sevenEighths, A: opaque},
		dimColor:    color.RGBA{R: half, G: half, B: half, A: opaque},
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
		brightColor: color.RGBA{R: tween, G: tween, B: sevenEighths, A: opaque},
		medColor:    color.RGBA{R: sevenEighths, G: sevenEighths, B: half, A: opaque},
		dimColor:    color.RGBA{R: half, G: half, B: quarter / two, A: opaque},
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
		medColor:    color.RGBA{R: sevenEighths, G: sevenEighths, B: 0, A: opaque},
		dimColor:    color.RGBA{R: half, G: half, B: 0, A: opaque},
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
		brightColor: color.RGBA{R: tween, G: tween - eighth, B: tween - quarter, A: opaque},
		medColor:    color.RGBA{R: threeQuarters, G: threeQuarters - eighth, B: half, A: opaque},
		dimColor:    color.RGBA{R: half, G: half - eighth, B: quarter, A: opaque},
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
		brightColor: color.RGBA{R: tween, G: 0, B: 0, A: opaque},
		medColor:    color.RGBA{R: sevenEighths, G: 0, B: 0, A: opaque},
		dimColor:    color.RGBA{R: threeQuarters, G: 0, B: 0, A: opaque},
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
	// classByZoom        = [11]int{7, 7, 7, 7, 7, 7, 6, 5, 4, 3, 2}
)

func getStarDetails(classDetails classDetails, sector sector, random1m *rand.Rand) []*star {
	stars := make([]*star, 0)
	loopSize := int32(500 * (classDetails.odds - classDetails.fudge + 2*classDetails.fudge*random1m.Float32()))
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
		nextStar.radii = (classDetails.minRadii + random1*classDetails.deltaRadii) / 2
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
		// if classCount > classByZoom[zoomIndex] {
		//  	break
		//}
	}

	return result
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

var stars []*star

var (
	sName       = "sphere"
	sphereModel *gi3d.Sphere
)

var (
	rendered      = false
	connectedStar int
	highWater     int
)

func renderStars(sc *gi3d.Scene) {
	if !rendered {
		stars = make([]*star, 0)
		id := 0
		for x := uint32(0); x < 4; x++ {
			for y := uint32(0); y < 2; y++ {
				sector := sector{x: x, y: y, z: 0}
				for _, star := range getSectorDetails(sector) {
					star.id = id
					id++
					stars = append(stars, star)
				}
			}
		}
		if len(stars) > 0 {
			sphereModel = &gi3d.Sphere{}
			sphereModel.Reset()
			sphereModel = gi3d.AddNewSphere(sc, sName, 0.002, 24)
			lines := make([]*simpleLine, 0)
			sName = "sphere"
			for _, star := range stars {
				starSphere := gi3d.AddNewSolid(sc, sc, sName, sphereModel.Name())
				starSphere.Pose.Pos.Set(star.x+offsets.x, star.y+offsets.y, star.z+offsets.z)
				starSphere.Mat.Color.SetUInt8(star.brightColor.R, star.brightColor.G, star.brightColor.B, star.brightColor.A)
			}
			for id, star := range stars {
				for _, jump := range checkForJumps(stars, star, id) {
					lines = append(lines, jump)
					if jump.jumpInfo.distance < 3 {
						jumpsByStar[star.id] = append(jumpsByStar[star.id], jump.jumpInfo)
					}
				}
			}

			rendered = true
			highWater = -1
			for lNumber := 0; lNumber < len(stars); lNumber++ {
				tJumps := traceJumps(lNumber)
				if len(tJumps) > highWater {
					highWater = len(tJumps)
					connectedStar = lNumber
				}
			}

			traceJumps := traceJumps(connectedStar)
			brighter := uint8(0)
			thicker := float32(1.0)

			popMax := 0
			bigWorld := worldFromStar(stars[0].id)
			bigStar := *stars[0]
			techMax := 0
			techWorld := worldFromStar(stars[0].id)
			techStar := *stars[0]

			for _, star := range stars {
				world := worldFromStar(star.id)
				if world.techLevelBase > techMax {
					techMax = world.techLevelBase
					techWorld = world
					techStar = *star
				}
				if world.popBase > popMax {
					popMax = world.popBase
					bigWorld = world
					bigStar = *star
				}
			}

			if techMax > popMax {
				techMax += 1
			}
			if bigWorld.popBase > popMax {
				techMax += 1
			}
			if bigStar.pixels > 0 {
				techMax += 1
			}
			if techWorld.popBase > popMax {
				techMax += 1
			}
			if techStar.pixels > 0 {
				techMax += 1
			}
			for id, lin := range lines {
				brighter = 0
				thicker = float32(1.0)
				for _, eachJump := range traceJumps {
					if lin.jumpInfo == eachJump {
						brighter = eighth
						thicker = float32(10.0)
					}
				}

				lin.jumpInfo.color.R += brighter
				lin.jumpInfo.color.G += brighter
				lin.jumpInfo.color.B += brighter
				thickness := float32(0.0002) * thicker

				if lin.jumpInfo.color.A < math.MaxUint8-55 {
					thickness = 0.00010 * thicker
				} else if lin.jumpInfo.color.A < math.MaxUint8-47 {
					thickness = 0.00012 * thicker
				} else if lin.jumpInfo.color.A < math.MaxUint8-39 {
					thickness = 0.00015 * thicker
				}
				jumpLines := gi3d.AddNewLines(sc, "Lines-"+strconv.Itoa(lin.jumpInfo.s1ID)+"-"+strconv.Itoa(lin.jumpInfo.s2ID),
					[]mat32.Vec3{
						{X: lin.from.x + offsets.x, Y: lin.from.y + offsets.y, Z: lin.from.z + offsets.z},
						{X: lin.to.x + offsets.x, Y: lin.to.y + offsets.y, Z: lin.to.z + offsets.z},
					},
					mat32.Vec2{X: thickness, Y: thickness},
					gi3d.OpenLines,
				)
				solidLine := gi3d.AddNewSolid(sc, sc, "Lines-"+strconv.Itoa(id), jumpLines.Name())
				// solidLine.Pose.Pos.Set(lin.from.x - .5, lin.from.y - .5, lin.from.z + 8)
				// lns.Mat.Color.SetUInt8(255, 255, 0, 128)
				solidLine.Mat.Color = lin.jumpInfo.color
			}
		}
	}
	print("Done")
}

func checkForJumps(stars []*star, star *star, id int) (result []*simpleLine) {
	result = make([]*simpleLine, 0)
	for innerId, innerStar := range stars {
		if innerId == id {
			continue
		}
		jumpColor := checkFor1jump(star, innerStar)
		if jumpColor.color.A > 0 {
			newJump := &simpleLine{
				from:     position{x: star.x, y: star.y, z: star.z},
				to:       position{x: innerStar.x, y: innerStar.y, z: innerStar.z},
				jumpInfo: jumpColor,
			}
			result = append(result, newJump)
		}
	}
	closest := []*simpleLine{&noLine, &noLine}
	if len(result) > 2 {
		for _, nextSimpleLine := range result {
			if nextSimpleLine.jumpInfo.distance < closest[0].jumpInfo.distance {
				if closest[0].jumpInfo.distance < closest[1].jumpInfo.distance {
					closest[1] = closest[0]
				}
				closest[0] = nextSimpleLine
			} else if nextSimpleLine.jumpInfo.distance < result[1].jumpInfo.distance {
				closest[1] = nextSimpleLine
			}
		}
		if closest[0].jumpInfo.parsecs > 3 || closest[1].jumpInfo.parsecs > 4 {
			result = append(result, closest[0])
		} else {
			result = closest
		}
	}
	return
}

func checkFor1jump(s1 *star, s2 *star) (result *jump) {
	jumpLength := distance(s1, s2) * 100 * parsecsPerLightYear
	delta := int(jumpLength)
	if delta < len(jumpColors) {
		if s1.id != s2.id {
			result = &jump{jumpColors[delta], delta, jumpLength, s1.id, s2.id}
		} else {
			result = &noJump
		}
	} else {
		result = &noJump
	}
	// Return transparent black if there isn't one

	return
}

func addVisits(base []*jump, addition []*jump) (result []*jump) {
	result = base
	for _, nextJump := range addition {
		already := false
		for _, baseJump := range base {
			if (baseJump.s1ID == nextJump.s1ID &&
				baseJump.s2ID == nextJump.s2ID) ||
				(baseJump.s1ID == nextJump.s2ID &&
					baseJump.s2ID == nextJump.s1ID) {
				already = true
				break
			}
		}
		if !already {
			result = append(result, nextJump)
		}
	}

	return
}

func subtractVisits(base []*jump, subtraction []*jump) (result []*jump) {
	result = make([]*jump, 0)
	for _, baseJump := range base {
		if baseJump.s1ID != baseJump.s2ID {
			add := true
			for _, nextJump := range subtraction {
				if nextJump.s1ID != nextJump.s2ID {
					if (baseJump.s1ID == nextJump.s1ID &&
						baseJump.s2ID == nextJump.s2ID) ||
						(baseJump.s1ID == nextJump.s2ID &&
							baseJump.s2ID == nextJump.s1ID) {
						add = false
						break
					}
				} else {
					continue
				}
			}
			if add {
				result = append(result, baseJump)
			}
		}
	}
	return
}

func nextVisits(base []*jump, visited []*jump) (result []*jump) {
	result = make([]*jump, 0)
	for _, start := range base {
		for _, nextJump := range jumpsByStar[start.s1ID] {
			result, _ = maybeAppend(result, nextJump)
		}
		for _, nextJump := range jumpsByStar[start.s2ID] {
			result, _ = maybeAppend(result, nextJump)
		}
	}
	result = subtractVisits(result, visited)
	return
}

func maybeAppend(soFar []*jump, nextJump *jump) (result []*jump, yesAppend bool) {
	yesAppend = true
	result = soFar
	for _, alreadyJumps := range soFar {
		if (nextJump.s1ID == alreadyJumps.s1ID && nextJump.s2ID == alreadyJumps.s2ID) ||
			(nextJump.s2ID == alreadyJumps.s1ID && nextJump.s1ID == alreadyJumps.s2ID) {
			yesAppend = false
			break
		}
	}
	if yesAppend {
		result = append(result, nextJump)
	}
	return
}

func traceJumps(id int) (visited []*jump) {
	explore := jumpsByStar[id]
	visited = explore
	for longest := 0; longest < 48; longest++ {
		if len(explore) == 0 {
			break
		}
		explore = nextVisits(explore, visited)
		visited = addVisits(visited, explore)
	}
	return visited
}

func showStar(star star, sc *gi3d.Scene) {
	starSphere := gi3d.AddNewSolid(sc, sc, sName, sphereModel.Name())
	starSphere.Pose.Pos.Set(star.x+offsets.x, star.y+offsets.y, star.z+offsets.z)
	starSphere.Mat.Color.SetUInt8(star.brightColor.R, star.brightColor.G, star.brightColor.B, star.brightColor.A)
}

func showBigStar(star star, sc *gi3d.Scene) {
	starSphere := gi3d.AddNewSolid(sc, sc, sName, sphereModel.Name())
	starSphere.Pose.Pos.Set(star.x+offsets.x, star.y+offsets.y, star.z+offsets.z)
	starSphere.Mat.Color.SetUInt8(star.brightColor.R, star.brightColor.G, star.brightColor.B, star.brightColor.A)
}


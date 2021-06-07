package main

import (
	"encoding/binary"
	"image/color"
	"math"
	"math/rand"
	"os"
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
	color       gist.Color
	activeColor gist.Color
	parsecs     int
	distance    float32
	s1ID        int
	s2ID        int
}

type simpleLine struct {
	from     position
	to       position
	jumpInfo *jump
	lines    *gi3d.Lines
}

const (
	intensityStep = 8
	faster        = true
	fastest       = false
)

var (
	size       = position{x: 2, y: 2, z: 1}
	sizeFloats = position{x: float32(size.x), y: float32(size.y), z: float32(size.z)}
	offsets    = position{x: sizeFloats.x / -2.0, y: sizeFloats.y / -2.0, z: sizeFloats.z / -2.0}

	intensity = []uint8{
		0, 0, intensityStep, 2 * intensityStep, 3 * intensityStep, 4 * intensityStep, 5 * intensityStep,
	}
	jumpColors = []gist.Color{
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: 0, B: 0, A: math.MaxUint8 - intensity[0]}),
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: half + eighth - eighth, B: 0, A: math.MaxUint8 - intensity[1]}),
		gist.Color(color.RGBA{R: math.MaxUint8 - eighth, G: math.MaxUint8 - eighth, B: 0, A: math.MaxUint8 - intensity[2]}),
		gist.Color(color.RGBA{R: 0, G: math.MaxUint8 - eighth, B: 0, A: math.MaxUint8 - intensity[3]}),
		gist.Color(color.RGBA{R: 0, G: 0, B: math.MaxUint8 - eighth, A: math.MaxUint8 - intensity[4]}),
		//gist.Color(color.RGBA{R: math.MaxUint8 - quarter, G: 0, B: math.MaxUint8 - quarter, A: math.MaxUint8 - intensity[5]}),//
	}

	tween  = uint8(sevenEighths)
	med    = uint8(threeQuarters)
	dim    = uint8(half)
	noJump = jump{color: gist.Color(color.RGBA{R: 0, G: 0, B: 0, A: 0}), activeColor: gist.Color(color.RGBA{R: 0, G: 0, B: 0, A: 0}),
		parsecs: 0, distance: 20480.0, s1ID: -1, s2ID: -1}
	noLine = simpleLine{from: position{x: 0, y: 0, z: 0}, to: position{x: 0, y: 0, z: 0}, jumpInfo: &noJump, lines: &gi3d.Lines{}}

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
var starPositionsDetails = []positionDetail {
	{ starposition: position{x: 0.3000606633, y: 2.999141706, z: -17.95709916}, starDetails:classG},
	{ starposition: position{x: -1.002472935, y: -1.377180927, z: -9.864011411}, starDetails:class},
	{ starposition: position{x: 3.096698006, y: 0.219086233, z: -17.34237086}, starDetails:classG},
	{ starposition: position{x: -3.793333946, y: -1.63617278, z: -18.7477786}, starDetails:classG},
	{ starposition: position{x: 1.638743605, y: 0.1842009541, z: -7.290788694}, starDetails:classG},
	{ starposition: position{x: -2.502102296, y: 3.622906187, z: -18.95048551}, starDetails:classF},
	{ starposition: position{x: 2.520221771, y: -0.0679129872, z: -9.841717125}, starDetails:classM},
	{ starposition: position{x: -0.3187608724, y: 2.150009068, z: -8.484113973}, starDetails:class},
	{ starposition: position{x: -0.3051047222, y: 2.05423386, z: -8.103515489}, starDetails:class},
	{ starposition: position{x: 2.08009681, y: -0.5984400014, z: -8.34300529}, starDetails:classK},
	{ starposition: position{x: 3.93587698, y: -1.712644077, z: -16.03572701}, starDetails:class},
	{ starposition: position{x: -0.1191676314, y: 2.66618989, z: -9.790936958}, starDetails:classG},
	{ starposition: position{x: -4.078426283, y: 1.80740184, z: -15.58964334}, starDetails:classF},
	{ starposition: position{x: 3.988197621, y: -3.675095533, z: -17.93142562}, starDetails:classG},
	{ starposition: position{x: -1.435729841, y: -4.446679544, z: -15.25975153}, starDetails:classM},
	{ starposition: position{x: 3.372796421, y: -0.3007396269, z: -10.88539024}, starDetails:classK},
	{ starposition: position{x: 4.169962804, y: -2.679951106, z: -15.34845119}, starDetails:classM},
	{ starposition: position{x: -1.60344971, y: -3.798826339, z: -11.38115267}, starDetails:classF},
	{ starposition: position{x: -5.797882053, y: -0.8545462451, z: -15.07423266}, starDetails:classK},
	{ starposition: position{x: 2.496379878, y: 2.736663654, z: -9.452387811}, starDetails:classD},
	{ starposition: position{x: 5.871039028, y: 0.05482463261, z: -14.73908899}, starDetails:classM},
	{ starposition: position{x: 6.037697061, y: 0.1395927117, z: -14.82196501}, starDetails:classK},
	{ starposition: position{x: 5.573255059, y: -1.243455082, z: -13.94550173}, starDetails:classK},
	{ starposition: position{x: 5.811813628, y: -4.86916525, z: -18.44247297}, starDetails:class},
	{ starposition: position{x: 2.975818544, y: 0.943581579, z: -7.516854697}, starDetails:classK},
	{ starposition: position{x: 3.556973853, y: -5.831561594, z: -16.34486698}, starDetails:classG},
	{ starposition: position{x: 3.634063857, y: 0.3184491451, z: -7.779713519}, starDetails:classF},
	{ starposition: position{x: 1.313522578, y: -2.087905356, z: -5.587332573}, starDetails:classG},
	{ starposition: position{x: -3.577522261, y: 6.041822544, z: -15.78596738}, starDetails:classK},
	{ starposition: position{x: 7.188046286, y: 0.1981136205, z: -16.02952085}, starDetails:classM},
	{ starposition: position{x: 6.037881712, y: -2.760633355, z: -14.58574846}, starDetails:classM},
	{ starposition: position{x: 0.8440926113, y: 6.151322952, z: -13.59221952}, starDetails:classK},
	{ starposition: position{x: 5.791020072, y: -2.151462104, z: -13.47979008}, starDetails:class},
	{ starposition: position{x: 3.010871616, y: -2.385499109, z: -8.377921866}, starDetails:classF},
	{ starposition: position{x: -5.265760905, y: -4.517745356, z: -14.86173726}, starDetails:classF},
	{ starposition: position{x: -1.960740582, y: 0.1228921667, z: -4.182666749}, starDetails:classD},
	{ starposition: position{x: -8.437229272, y: -0.2533913492, z: -17.78770178}, starDetails:classF},
	{ starposition: position{x: 3.725642811, y: 3.484809597, z: -10.31345281}, starDetails:classM},
	{ starposition: position{x: 4.524062982, y: 5.737680634, z: -14.65404737}, starDetails:classK},
	{ starposition: position{x: -2.853268952, y: -4.708839867, z: -11.00919014}, starDetails:classF},
	{ starposition: position{x: -2.813163512, y: 8.515701193, z: -17.86754421}, starDetails:classK},
	{ starposition: position{x: -7.314430266, y: -2.005663589, z: -15.01074683}, starDetails:class},
	{ starposition: position{x: -0.471753504, y: -0.3613218058, z: -1.150373428}, starDetails:classM},
	{ starposition: position{x: 3.629907325, y: 4.240368188, z: -10.75784469}, starDetails:classG},
	{ starposition: position{x: 3.617798708, y: 4.242851516, z: -10.71475753}, starDetails:classG},
	{ starposition: position{x: -0.6429634711, y: -2.189390971, z: -4.365468847}, starDetails:class},
	{ starposition: position{x: -1.191536398, y: -4.056917052, z: -8.088030193}, starDetails:classM},
	{ starposition: position{x: 8.733932911, y: 2.230621306, z: -17.13035955}, starDetails:classK},
	{ starposition: position{x: 1.707110509, y: -7.630571486, z: -14.73751405}, starDetails:classM},
	{ starposition: position{x: 0.3794700338, y: -8.326387762, z: -15.67773919}, starDetails:classG},
	{ starposition: position{x: -6.083437451, y: -4.046131928, z: -13.62551905}, starDetails:class},
	{ starposition: position{x: -6.048766291, y: 0.939576741, z: -11.34370582}, starDetails:classK},
	{ starposition: position{x: -2.284926607, y: 8.527829432, z: -16.15046254}, starDetails:classK},
	{ starposition: position{x: -0.5035983605, y: -0.4212846275, z: -1.176707216}, starDetails:classK},
	{ starposition: position{x: -0.5036202776, y: -0.4213968308, z: -1.176657659}, starDetails:classG},
	{ starposition: position{x: -1.443353387, y: -6.533626747, z: -11.93823922}, starDetails:classK},
	{ starposition: position{x: 9.31869427, y: -1.681238816, z: -16.75067002}, starDetails:classK},
	{ starposition: position{x: 1.358863278, y: 9.565460367, z: -17.01821607}, starDetails:classG},
	{ starposition: position{x: -3.943019316, y: 6.985487044, z: -14.06509997}, starDetails:classG},
	{ starposition: position{x: -4.005689603, y: 3.31252377, z: -9.106419596}, starDetails:classM},
	{ starposition: position{x: 8.544982806, y: 4.762564949, z: -17.10008707}, starDetails:classK},
	{ starposition: position{x: 0.1657858176, y: 6.407700532, z: -11.11262191}, starDetails:classK},
	{ starposition: position{x: 5.171675274, y: -2.999082658, z: -10.25228539}, starDetails:classM},
	{ starposition: position{x: -4.951742452, y: -3.438003261, z: -10.18388427}, starDetails:classK},
	{ starposition: position{x: 4.063180337, y: 8.375668415, z: -15.67957423}, starDetails:classK},
	{ starposition: position{x: -8.772384074, y: -2.946146937, z: -15.46387699}, starDetails:classF},
	{ starposition: position{x: 5.958810739, y: 4.906753324, z: -12.7223854}, starDetails:class},
	{ starposition: position{x: -1.253324427, y: 8.426331392, z: -13.91836179}, starDetails:classM},
	{ starposition: position{x: 6.08724614, y: -8.480266984, z: -16.89177677}, starDetails:classM},
	{ starposition: position{x: 6.697440295, y: 5.571186671, z: -14.04354874}, starDetails:classM},
	{ starposition: position{x: 8.49197345, y: -5.58907821, z: -16.08205579}, starDetails:classK},
	{ starposition: position{x: -3.294405727, y: -6.594924821, z: -11.60164854}, starDetails:classK},
	{ starposition: position{x: 1.449074839, y: 6.011574063, z: -9.726816641}, starDetails:classK},
	{ starposition: position{x: -6.607941065, y: 1.283013666, z: -10.58668942}, starDetails:classM},
	{ starposition: position{x: 1.475440193, y: 6.088451547, z: -9.823349122}, starDetails:classF},
	{ starposition: position{x: -0.1835050071, y: -3.132757141, z: -4.891093121}, starDetails:classM},
	{ starposition: position{x: 9.582550415, y: -4.289269491, z: -16.30314036}, starDetails:classK},
	{ starposition: position{x: 5.543845014, y: -5.389327383, z: -11.88953282}, starDetails:classM},
	{ starposition: position{x: 1.734635661, y: -0.9685547889, z: -3.033693328}, starDetails:classK},
	{ starposition: position{x: -5.391063447, y: -3.541545433, z: -9.835501259}, starDetails:classM},
	{ starposition: position{x: -3.672514126, y: -7.033511533, z: -11.96332034}, starDetails:classK},
	{ starposition: position{x: 4.109488733, y: 1.911646668, z: -6.769449364}, starDetails:classK},
	{ starposition: position{x: 1.754939139, y: -6.846198165, z: -10.47444945}, starDetails:classM},
	{ starposition: position{x: 3.48479726, y: -10.65769359, z: -16.54178112}, starDetails:classM},
	{ starposition: position{x: 1.794165909, y: 6.010535287, z: -9.250849512}, starDetails:classM},
	{ starposition: position{x: -1.27099662, y: 7.276083736, z: -10.79127986}, starDetails:classK},
	{ starposition: position{x: 7.062368838, y: -3.208345542, z: -10.89946699}, starDetails:classK},
	{ starposition: position{x: 1.585435004, y: -10.90550328, z: -15.31518756}, starDetails:classK},
	{ starposition: position{x: 9.178910403, y: 3.09654765, z: -13.30530789}, starDetails:classM},
	{ starposition: position{x: 9.253620335, y: 4.437854818, z: -13.99179805}, starDetails:classF},
	{ starposition: position{x: 7.289626917, y: -3.467099276, z: -10.95900634}, starDetails:classG},
	{ starposition: position{x: -9.247155899, y: -4.014229245, z: -13.58964694}, starDetails:class},
	{ starposition: position{x: 4.148621095, y: 7.905653043, z: -12.01075155}, starDetails:classM},
	{ starposition: position{x: 3.427497256, y: 7.02240147, z: -10.48767956}, starDetails:classK},
	{ starposition: position{x: -3.747079342, y: 0.8907549042, z: -5.161326058}, starDetails:class},
	{ starposition: position{x: 5.781193404, y: 5.305107889, z: -10.46573779}, starDetails:classM},
	{ starposition: position{x: -5.371535224, y: 8.433583349, z: -13.25449405}, starDetails:classM},
	{ starposition: position{x: 4.774305709, y: -5.574622339, z: -9.633502261}, starDetails:classK},
	{ starposition: position{x: -1.623855219, y: -7.477427725, z: -9.977645675}, starDetails:class},
	{ starposition: position{x: -9.193330612, y: -2.678326662, z: -12.45249678}, starDetails:classK},
	{ starposition: position{x: -5.864856288, y: -0.9784576646, z: -7.610850002}, starDetails:classM},
	{ starposition: position{x: -0.6525368156, y: -9.417499402, z: -12.01068265}, starDetails:classG},
	{ starposition: position{x: 8.097522842, y: 9.933958954, z: -15.27084867}, starDetails:classK},
	{ starposition: position{x: -1.298782271, y: -9.929099041, z: -12.65068202}, starDetails:classM},
	{ starposition: position{x: 9.498561509, y: 5.262278745, z: -13.70514735}, starDetails:classG},
	{ starposition: position{x: 0.6718166407, y: 12.09655214, z: -14.99668776}, starDetails:classA},
	{ starposition: position{x: 8.790249467, y: -5.194022438, z: -12.61080127}, starDetails:classM},
	{ starposition: position{x: -9.13259473, y: -4.886948649, z: -12.75524404}, starDetails:classK},
	{ starposition: position{x: 7.545045363, y: -5.447622572, z: -11.42873318}, starDetails:classK},
	{ starposition: position{x: 5.807770981, y: 3.714225526, z: -8.460369567}, starDetails:classK},
	{ starposition: position{x: 8.268951443, y: 7.096907128, z: -13.36126909}, starDetails:classG},
	{ starposition: position{x: 3.861805307, y: -6.51823799, z: -9.045723316}, starDetails:classK},
	{ starposition: position{x: 3.862485926, y: -6.518275862, z: -9.045405425}, starDetails:class},
	{ starposition: position{x: -1.569381179, y: 9.587658781, z: -11.59391797}, starDetails:classK},
	{ starposition: position{x: 8.020216708, y: 9.840995044, z: -15.12561556}, starDetails:classK},
	{ starposition: position{x: 2.599935855, y: -1.93146477, z: -3.726805576}, starDetails:classM},
	{ starposition: position{x: -14.4161946, y: 7.334230865, z: -5.448925427}, starDetails:classK},
	{ starposition: position{x: -0.6963014545, y: -6.416268146, z: -7.3417053}, starDetails:classK},
	{ starposition: position{x: 5.241914544, y: 7.149706295, z: -9.992365901}, starDetails:classK},
	{ starposition: position{x: -6.162837106, y: -7.464580234, z: -10.87096147}, starDetails:classG},
	{ starposition: position{x: 2.356049369, y: -8.92957939, z: -10.35559455}, starDetails:classM},
	{ starposition: position{x: -6.209681108, y: 2.819082803, z: -7.566743581}, starDetails:class},
	{ starposition: position{x: 9.885109188, y: 2.00024612, z: -11.0264831}, starDetails:classG},
	{ starposition: position{x: 8.899769326, y: -5.765225674, z: -11.49255503}, starDetails:classG},
	{ starposition: position{x: 8.813765117, y: -5.360975441, z: -11.05953775}, starDetails:classM},
	{ starposition: position{x: -3.343399815, y: 9.508270629, z: -10.80105637}, starDetails:classK},
	{ starposition: position{x: -0.422644453, y: -3.071188083, z: -3.312069185}, starDetails:classK},
	{ starposition: position{x: -9.459233556, y: 5.540532982, z: -11.52131054}, starDetails:classM},
	{ starposition: position{x: -1.072271647, y: -5.937023942, z: -6.387946846}, starDetails:classM},
	{ starposition: position{x: 2.412683841, y: -3.649345218, z: -4.399846916}, starDetails:classM},
	{ starposition: position{x: 7.077210888, y: -10.14297262, z: -13.00030831}, starDetails:classK},
	{ starposition: position{x: 13.78085277, y: 1.304211404, z: -14.38026136}, starDetails:classM},
	{ starposition: position{x: -12.42347112, y: -1.9662914, z: -12.99657087}, starDetails:classM},
	{ starposition: position{x: 3.60490531, y: -10.36463942, z: -11.31932337}, starDetails:classK},
	{ starposition: position{x: -5.684600652, y: 3.816159667, z: -7.034528551}, starDetails:classM},
	{ starposition: position{x: 12.47918261, y: 1.16458016, z: -12.86340146}, starDetails:classM},
	{ starposition: position{x: 9.982845038, y: 3.397277394, z: -10.74278169}, starDetails:classF},
	{ starposition: position{x: 0.5807830102, y: 2.708354337, z: -2.770360228}, starDetails:classM},
	{ starposition: position{x: 4.62176097, y: 6.185640646, z: -7.641747629}, starDetails:class},
	{ starposition: position{x: -7.093006241, y: -10.24283119, z: -12.31223056}, starDetails:classF},
	{ starposition: position{x: -8.578141731, y: 9.834755261, z: -12.81218761}, starDetails:classM},
	{ starposition: position{x: -8.909540908, y: 0.7383634485, z: -8.75654366}, starDetails:classK},
	{ starposition: position{x: -0.3603451746, y: -3.589695344, z: -3.522724882}, starDetails:classM},
	{ starposition: position{x: -1.433095747, y: 5.562220339, z: -5.60346706}, starDetails:classM},
	{ starposition: position{x: 6.494302449, y: 5.183188728, z: -7.966941767}, starDetails:classK},
	{ starposition: position{x: -3.009770085, y: 10.49918527, z: -10.40541224}, starDetails:classK},
	{ starposition: position{x: 7.707621497, y: -6.950744439, z: -9.887770336}, starDetails:classM},
	{ starposition: position{x: -3.240939605, y: 11.31056935, z: -11.20788244}, starDetails:classG},
	{ starposition: position{x: -5.463646011, y: -0.9826214177, z: -5.280537608}, starDetails:classM},
	{ starposition: position{x: -3.383295904, y: 11.83045604, z: -11.70220424}, starDetails:classK},
	{ starposition: position{x: -6.843321228, y: 4.313167378, z: -7.676762699}, starDetails:classK},
	{ starposition: position{x: 0.5068764326, y: -9.393548834, z: -8.909603099}, starDetails:classK},
	{ starposition: position{x: 2.847127339, y: 3.389681085, z: -4.138353661}, starDetails:classG},
	{ starposition: position{x: 1.846190291, y: 14.13421024, z: -13.15207371}, starDetails:classM},
	{ starposition: position{x: -5.490363889, y: -9.291326924, z: -9.932257172}, starDetails:classK},
	{ starposition: position{x: 11.68715038, y: 2.367200386, z: -10.702605}, starDetails:classK},
	{ starposition: position{x: 11.81213953, y: 4.390574236, z: -11.21011276}, starDetails:classK},
	{ starposition: position{x: 8.460967803, y: -5.432600707, z: -8.913950224}, starDetails:classM},
	{ starposition: position{x: 13.76544528, y: -1.308028823, z: -12.24090751}, starDetails:classM},
	{ starposition: position{x: 6.790757375, y: 11.36965675, z: -11.65242038}, starDetails:classK},
	{ starposition: position{x: -2.680417667, y: -3.564495989, z: -3.914364427}, starDetails:classM},
	{ starposition: position{x: -5.858776693, y: 4.110761471, z: -6.236565957}, starDetails:classM},
	{ starposition: position{x: -7.840609435, y: 0.9701912573, z: -6.879069534}, starDetails:classM},
	{ starposition: position{x: -7.013575989, y: 0.4124621753, z: -6.000792381}, starDetails:classG},
	{ starposition: position{x: -11.22627231, y: 8.559725544, z: -12.04320813}, starDetails:classF},
	{ starposition: position{x: 8.245639792, y: 10.14509711, z: -10.99965372}, starDetails:classK},
	{ starposition: position{x: -2.345875928, y: 14.6840909, z: -12.48063797}, starDetails:class},
	{ starposition: position{x: -3.883778136, y: -14.22044853, z: -12.1773174}, starDetails:classK},
	{ starposition: position{x: -4.054519801, y: -9.060067866, z: -8.103896467}, starDetails:classD},
	{ starposition: position{x: -4.056714865, y: -9.113733825, z: -8.134051564}, starDetails:classG},
	{ starposition: position{x: 3.560021608, y: -9.316809798, z: -8.131291766}, starDetails:class},
	{ starposition: position{x: -3.075141091, y: -10.86888278, z: -9.177556759}, starDetails:classM},
	{ starposition: position{x: 2.128131838, y: 13.62976135, z: -11.15874345}, starDetails:classK},
	{ starposition: position{x: -3.627650587, y: -10.57299256, z: -9.033538082}, starDetails:classK},
	{ starposition: position{x: -5.668393579, y: 6.560860399, z: -6.991924058}, starDetails:classK},
	{ starposition: position{x: 2.330654489, y: -2.002995569, z: -2.476543224}, starDetails:classM},
	{ starposition: position{x: -3.039439295, y: -14.12063461, z: -11.45214398}, starDetails:class},
	{ starposition: position{x: 7.115374353, y: 10.56391304, z: -10.05255453}, starDetails:classK},
	{ starposition: position{x: -10.8362601, y: 3.958175194, z: -8.987454918}, starDetails:class},
	{ starposition: position{x: -6.572326584, y: -10.07137283, z: -9.367418752}, starDetails:classG},
	{ starposition: position{x: 3.902517825, y: -12.99196632, z: -10.52612876}, starDetails:classG},
	{ starposition: position{x: -2.837691195, y: -6.090132684, z: -5.161369161}, starDetails:classM},
	{ starposition: position{x: 3.465359632, y: 0.08073110829, z: -2.645557385}, starDetails:classM},
	{ starposition: position{x: 0.6495652604, y: 12.05921035, z: -8.880388265}, starDetails:classM},
	{ starposition: position{x: 4.079982656, y: -13.68225149, z: -10.78349298}, starDetails:classF},
	{ starposition: position{x: 1.751177302, y: -6.325838056, z: -4.937569578}, starDetails:classA},
	{ starposition: position{x: -12.28180055, y: 4.400338482, z: -9.793637625}, starDetails:classM},
	{ starposition: position{x: 9.192204984, y: 6.385837511, z: -8.36690819}, starDetails:class},
	{ starposition: position{x: -13.52592608, y: -4.963247818, z: -10.74396463}, starDetails:classA},
	{ starposition: position{x: -12.36159011, y: 4.385662752, z: -9.753100531}, starDetails:classK},
	{ starposition: position{x: -12.80444047, y: -7.899599339, z: -11.07957837}, starDetails:classK},
	{ starposition: position{x: 2.648813692, y: -4.110411476, z: -3.565477299}, starDetails:classK},
	{ starposition: position{x: 0.3918448762, y: -14.04131611, z: -10.21300636}, starDetails:classG},
	{ starposition: position{x: 2.592687907, y: -0.6250093309, z: -1.927457451}, starDetails:classM},
	{ starposition: position{x: 12.64199271, y: 4.430613345, z: -9.631422354}, starDetails:class},
	{ starposition: position{x: 4.433066421, y: 14.0326659, z: -10.34557022}, starDetails:classK},
	{ starposition: position{x: -1.017739827, y: -5.619679685, z: -3.997402114}, starDetails:classK},
	{ starposition: position{x: -10.95868864, y: 5.579326659, z: -0.576323894}, starDetails:classM},
	{ starposition: position{x: -6.971504781, y: 13.07522087, z: -10.26254361}, starDetails:classF},
	{ starposition: position{x: -11.63555133, y: -5.491846769, z: -8.830910505}, starDetails:classK},
	{ starposition: position{x: -5.590622748, y: 11.26392727, z: -8.538453211}, starDetails:classG},
	{ starposition: position{x: -7.467127762, y: 13.22412181, z: -10.22540427}, starDetails:classK},
	{ starposition: position{x: 16.28669958, y: 2.982701294, z: -11.0112201}, starDetails:class},
	{ starposition: position{x: -14.46131178, y: -7.18595574, z: -10.50421214}, starDetails:classF},
	{ starposition: position{x: -7.965043692, y: 0.8899314761, z: -5.171664138}, starDetails:classK},
	{ starposition: position{x: 15.65509285, y: -1.379723641, z: -10.1058821}, starDetails:classK},
	{ starposition: position{x: 14.01675668, y: -7.285674337, z: -10.08269223}, starDetails:classF},
	{ starposition: position{x: -7.575270666, y: 0.8145796328, z: -4.860889085}, starDetails:classM},
	{ starposition: position{x: 2.219712483, y: 14.96084951, z: -9.637132219}, starDetails:classK},
	{ starposition: position{x: 5.600378276, y: -6.565924112, z: -5.483975129}, starDetails:classM},
	{ starposition: position{x: -12.322743, y: -6.258561887, z: -8.781209344}, starDetails:classK},
	{ starposition: position{x: -7.679846184, y: 5.85319672, z: 0.0546049452}, starDetails:classK},
	{ starposition: position{x: 5.564875204, y: -1.302016111, z: -3.608537467}, starDetails:class},
	{ starposition: position{x: -0.3148626551, y: -5.226948499, z: -3.285155357}, starDetails:classM},
	{ starposition: position{x: -13.0166305, y: 9.945766522, z: -10.27770656}, starDetails:classK},
	{ starposition: position{x: -0.3141805547, y: -5.227053672, z: -3.285053319}, starDetails:classM},
	{ starposition: position{x: -4.387172919, y: -15.20779108, z: -9.916120721}, starDetails:classK},
	{ starposition: position{x: 8.752340008, y: 5.78673741, z: -6.566535842}, starDetails:class},
	{ starposition: position{x: -11.27097053, y: 3.810707486, z: -7.094426844}, starDetails:class},
	{ starposition: position{x: 6.567514995, y: -10.53615204, z: -7.683544698}, starDetails:classM},
	{ starposition: position{x: 6.258165186, y: -1.782770555, z: -3.997771311}, starDetails:classK},
	{ starposition: position{x: -6.420840238, y: 8.154235887, z: -6.360788886}, starDetails:classK},
	{ starposition: position{x: 15.47248955, y: 1.485290537, z: -9.489944484}, starDetails:classM},
	{ starposition: position{x: 5.602748533, y: -6.379963149, z: -5.170651848}, starDetails:classM},
	{ starposition: position{x: 7.98482143, y: -14.31465072, z: -9.979789766}, starDetails:classK},
	{ starposition: position{x: -12.35446248, y: -7.943710206, z: -8.927112923}, starDetails:class},
	{ starposition: position{x: -0.01846797842, y: 12.82379528, z: -7.714822967}, starDetails:classK},
	{ starposition: position{x: 10.10950376, y: 6.862602879, z: -7.338562013}, starDetails:class},
	{ starposition: position{x: -6.410394086, y: -13.67303123, z: -9.038541973}, starDetails:classG},
	{ starposition: position{x: 10.01536252, y: -13.32255038, z: -9.961954056}, starDetails:classG},
	{ starposition: position{x: -12.02076133, y: 6.857119816, z: -8.116621679}, starDetails:classM},
	{ starposition: position{x: 11.82808231, y: 2.785793326, z: -7.117112836}, starDetails:classK},
	{ starposition: position{x: -15.32503895, y: 0.8523450486, z: -8.964024128}, starDetails:classG},
	{ starposition: position{x: -6.667047243, y: 2.798060902, z: 0.1063614701}, starDetails:classM},
	{ starposition: position{x: -5.084139464, y: 12.44935015, z: -7.84123878}, starDetails:classM},
	{ starposition: position{x: 7.18277205, y: -11.17208579, z: -7.735781802}, starDetails:class},
	{ starposition: position{x: 6.980806428, y: 4.270102781, z: -4.758342927}, starDetails:classK},
	{ starposition: position{x: 12.33988428, y: -5.158630108, z: -7.727036621}, starDetails:classK},
	{ starposition: position{x: 11.06814174, y: -3.989206104, z: -6.703738433}, starDetails:classK},
	{ starposition: position{x: 6.43705109, y: -1.795831232, z: -3.799763371}, starDetails:classA},
	{ starposition: position{x: -12.85962678, y: -7.05836807, z: -8.161612105}, starDetails:classK},
	{ starposition: position{x: 2.779645982, y: 6.689887224, z: -4.019983929}, starDetails:class},
	{ starposition: position{x: 8.257043901, y: 9.176118283, z: -6.839448464}, starDetails:classF},
	{ starposition: position{x: 16.31575189, y: -4.925568234, z: -9.367363452}, starDetails:classM},
	{ starposition: position{x: -3.80975205, y: -15.41269868, z: -8.649726617}, starDetails:classK},
	{ starposition: position{x: 4.368732082, y: 15.15976423, z: -8.588653094}, starDetails:classK},
	{ starposition: position{x: -13.31540733, y: -5.232941227, z: -7.727169049}, starDetails:classM},
	{ starposition: position{x: 5.238610298, y: 5.616039811, z: -4.121618472}, starDetails:class},
	{ starposition: position{x: 11.47626635, y: 12.29956623, z: -9.027294437}, starDetails:classS},
	{ starposition: position{x: 11.20968648, y: 2.922584584, z: -6.122323129}, starDetails:class},
	{ starposition: position{x: -15.25699594, y: 2.888158037, z: -8.191585099}, starDetails:classM},
	{ starposition: position{x: 8.188573098, y: -10.89659565, z: -7.169970376}, starDetails:classM},
	{ starposition: position{x: -9.019083541, y: 0.08072945684, z: -4.73642621}, starDetails:classK},
	{ starposition: position{x: -8.718289895, y: 5.269651488, z: -5.249180848}, starDetails:class},
	{ starposition: position{x: 15.18070128, y: -1.053424485, z: -7.819345472}, starDetails:classG},
	{ starposition: position{x: 4.37453389, y: -6.529897007, z: -4.010380217}, starDetails:classK},
	{ starposition: position{x: 15.95063293, y: 1.706387555, z: -8.183143424}, starDetails:classK},
	{ starposition: position{x: 3.357961927, y: -15.03675814, z: -7.824462814}, starDetails:classK},
	{ starposition: position{x: 0.2431066799, y: 13.0064905, z: -6.587302678}, starDetails:class},
	{ starposition: position{x: -15.11979258, y: -4.217721786, z: -7.924548254}, starDetails:classK},
	{ starposition: position{x: -1.035928616, y: -5.250430521, z: -2.679912927}, starDetails:classK},
	{ starposition: position{x: -1.013508866, y: -5.241877965, z: -2.666960058}, starDetails:classK},
	{ starposition: position{x: 10.84994462, y: 12.34386408, z: -8.174775429}, starDetails:classK},
	{ starposition: position{x: 12.84369757, y: -10.79943769, z: -8.312449542}, starDetails:classG},
	{ starposition: position{x: -2.990242582, y: -2.729145152, z: -1.984284735}, starDetails:classM},
	{ starposition: position{x: -2.819369716, y: -2.573503522, z: -1.870573614}, starDetails:classM},
	{ starposition: position{x: -3.469337758, y: 12.67594455, z: -6.395323718}, starDetails:classK},
	{ starposition: position{x: 9.367091319, y: 6.496816331, z: -5.546854812}, starDetails:classG},
	{ starposition: position{x: 13.78027351, y: -3.038951969, z: -6.861669172}, starDetails:class},
	{ starposition: position{x: -13.15312591, y: -9.12624362, z: -7.74475339}, starDetails:classF},
	{ starposition: position{x: -0.346628808, y: 10.27189085, z: -4.956147039}, starDetails:class},
	{ starposition: position{x: -8.09001774, y: 14.28902863, z: -7.876442139}, starDetails:classK},
	{ starposition: position{x: -8.444443298, y: 15.52289822, z: -8.354488648}, starDetails:classM},
	{ starposition: position{x: 8.795375695, y: -9.932967845, z: -6.263095442}, starDetails:classF},
	{ starposition: position{x: 13.15398333, y: -5.180880039, z: -6.665169845}, starDetails:class},
	{ starposition: position{x: -17.6454015, y: -3.765277826, z: -8.339558656}, starDetails:classK},
	{ starposition: position{x: -16.65911835, y: -3.553917463, z: -7.872606461}, starDetails:classK},
	{ starposition: position{x: 13.91734684, y: 2.287146323, z: -6.507187691}, starDetails:classK},
	{ starposition: position{x: 17.59728387, y: -4.042154829, z: -8.321024084}, starDetails:classK},
	{ starposition: position{x: -13.40549881, y: -0.4922920611, z: -6.178162293}, starDetails:classF},
	{ starposition: position{x: -6.267125897, y: -13.933777, z: -7.029871164}, starDetails:classM},
	{ starposition: position{x: -9.483713804, y: 2.122861987, z: -4.44893159}, starDetails:classM},
	{ starposition: position{x: -16.04322214, y: -8.471607843, z: -8.225933503}, starDetails:classG},
	{ starposition: position{x: -9.259819486, y: 14.8844778, z: -7.916617292}, starDetails:classF},
	{ starposition: position{x: -11.31948791, y: -10.52649669, z: -6.980383449}, starDetails:classK},
	{ starposition: position{x: 16.34065349, y: 7.042518258, z: -7.988414151}, starDetails:classK},
	{ starposition: position{x: 6.214809693, y: -11.41095879, z: -5.768850527}, starDetails:classK},
	{ starposition: position{x: -11.22068983, y: 7.694152563, z: -6.033639055}, starDetails:classG},
	{ starposition: position{x: -16.12904477, y: 1.792126172, z: -7.181283982}, starDetails:class},
	{ starposition: position{x: -0.9157936276, y: 15.23997132, z: -6.753563495}, starDetails:classG},
	{ starposition: position{x: -2.233413261, y: -16.4712287, z: -7.343613995}, starDetails:classK},
	{ starposition: position{x: 0.5862507907, y: -2.654493843, z: -1.201011484}, starDetails:classM},
	{ starposition: position{x: 8.623619789, y: -10.25988845, z: -5.904303304}, starDetails:classG},
	{ starposition: position{x: 9.0388297, y: 13.76682351, z: -7.075081701}, starDetails:classF},
	{ starposition: position{x: 16.90987437, y: 3.705550931, z: -7.424125402}, starDetails:classG},
	{ starposition: position{x: 13.64395959, y: 3.12241073, z: -5.915134513}, starDetails:classK},
	{ starposition: position{x: -11.7626042, y: 14.12612366, z: -7.675472025}, starDetails:classG},
	{ starposition: position{x: 7.268164358, y: -1.938045818, z: -3.119529806}, starDetails:classK},
	{ starposition: position{x: 0.5614424123, y: 8.270766525, z: -3.424849828}, starDetails:classF},
	{ starposition: position{x: 9.033834526, y: 4.844440154, z: -4.232369228}, starDetails:classK},
	{ starposition: position{x: -11.66436784, y: -6.056900522, z: -5.33954036}, starDetails:classK},
	{ starposition: position{x: 15.12509484, y: 6.454744256, z: -6.611765861}, starDetails:class},
	{ starposition: position{x: -6.679120309, y: -15.21956587, z: -6.693772628}, starDetails:classK},
	{ starposition: position{x: -0.2472537493, y: 5.353017441, z: -2.150164769}, starDetails:classM},
	{ starposition: position{x: -1.177582427, y: -16.25989973, z: -6.481972826}, starDetails:classF},
	{ starposition: position{x: -6.749507645, y: 4.902641936, z: -3.312736132}, starDetails:classK},
	{ starposition: position{x: -4.646780723, y: 15.34731162, z: -6.30261995}, starDetails:classM},
	{ starposition: position{x: -3.930853652, y: -3.844536244, z: -2.15603645}, starDetails:classK},
	{ starposition: position{x: -4.981344493, y: -4.870947953, z: -2.731408708}, starDetails:classK},
	{ starposition: position{x: 1.972016052, y: 7.690966792, z: -3.088578601}, starDetails:classM},
	{ starposition: position{x: -4.222185642, y: 9.192585961, z: -3.928439453}, starDetails:classM},
	{ starposition: position{x: -2.748224146, y: -15.99518791, z: -6.266514961}, starDetails:classF},
	{ starposition: position{x: 2.046331465, y: -11.94304345, z: -4.663814344}, starDetails:classG},
	{ starposition: position{x: 7.585647931, y: -2.80771306, z: -3.043675822}, starDetails:classM},
	{ starposition: position{x: 7.904084171, y: -12.36747362, z: -5.485506459}, starDetails:classK},
	{ starposition: position{x: -12.15430818, y: 2.064336939, z: -4.597988647}, starDetails:classK},
	{ starposition: position{x: -8.312059685, y: -15.25671022, z: -6.478333828}, starDetails:classK},
	{ starposition: position{x: 13.75519176, y: -1.797431134, z: -5.156371866}, starDetails:classM},
	{ starposition: position{x: 14.14063771, y: 10.75491996, z: -6.459089756}, starDetails:classK},
	{ starposition: position{x: -12.01444359, y: -0.5875093866, z: -4.368647564}, starDetails:class},
	{ starposition: position{x: 11.81421012, y: -9.892924093, z: -5.563033981}, starDetails:classK},
	{ starposition: position{x: 7.313554456, y: 9.349808263, z: -4.274837953}, starDetails:classK},
	{ starposition: position{x: 17.94543007, y: -3.628816841, z: -6.535416227}, starDetails:classK},
	{ starposition: position{x: -4.378627972, y: -6.378673832, z: -2.735112656}, starDetails:classM},
	{ starposition: position{x: -17.10066771, y: 3.990139134, z: -6.146462261}, starDetails:classK},
	{ starposition: position{x: -2.984771552, y: 18.48817301, z: -6.542049555}, starDetails:classK},
	{ starposition: position{x: -12.70610217, y: 8.683817297, z: -5.369695329}, starDetails:class},
	{ starposition: position{x: 7.427973261, y: 11.00405205, z: -4.600308089}, starDetails:classK},
	{ starposition: position{x: 12.30493441, y: -14.28734319, z: -6.461750765}, starDetails:classM},
	{ starposition: position{x: 1.702184336, y: -12.3790701, z: -4.280215369}, starDetails:classK},
	{ starposition: position{x: -6.361024761, y: -12.57850133, z: -4.819073859}, starDetails:class},
	{ starposition: position{x: 9.955456535, y: 8.738364959, z: -4.450906523}, starDetails:classF},
	{ starposition: position{x: -7.624436229, y: -2.715546909, z: -2.678048897}, starDetails:classG},
	{ starposition: position{x: -8.391290901, y: -0.9139507069, z: -2.781221136}, starDetails:classM},
	{ starposition: position{x: 1.970142857, y: 8.633063315, z: -2.906011213}, starDetails:classM},
	{ starposition: position{x: -8.549354068, y: -12.27512078, z: -4.874674549}, starDetails:classK},
	{ starposition: position{x: -9.706753189, y: -4.831593557, z: -3.516097522}, starDetails:classM},
	{ starposition: position{x: 3.352007073, y: 11.28522781, z: -3.773365316}, starDetails:classM},
	{ starposition: position{x: 7.690056296, y: 4.669532902, z: -2.85639293}, starDetails:classM},
	{ starposition: position{x: 2.165437219, y: 8.579297015, z: -2.768236332}, starDetails:classK},
	{ starposition: position{x: 3.378005917, y: 1.105707556, z: -1.086715149}, starDetails:classM},
	{ starposition: position{x: 3.590299341, y: -3.831533577, z: -1.602806947}, starDetails:class},
	{ starposition: position{x: 13.21619112, y: -12.6629178, z: -5.579840351}, starDetails:classM},
	{ starposition: position{x: 3.957262001, y: 12.10789122, z: -3.878620606}, starDetails:classG},
	{ starposition: position{x: 13.25479477, y: -1.579374771, z: -4.041638139}, starDetails:classK},
	{ starposition: position{x: -0.4943985262, y: 2.476800793, z: -0.7583666751}, starDetails:classA},
	{ starposition: position{x: 7.229644048, y: 7.347962501, z: -3.071583761}, starDetails:class},
	{ starposition: position{x: -17.31472083, y: -2.439184148, z: -5.078692843}, starDetails:classF},
	{ starposition: position{x: 4.995504627, y: 0.3375182732, z: -1.448224492}, starDetails:classM},
	{ starposition: position{x: 9.499482302, y: -6.225876424, z: -3.283992687}, starDetails:classA},
	{ starposition: position{x: 3.151620992, y: 1.538603258, z: -1.001652705}, starDetails:classG},
	{ starposition: position{x: -5.343446715, y: -16.63442874, z: -4.923680199}, starDetails:class},
	{ starposition: position{x: 18.18320107, y: 0.8944872152, z: -5.037566142}, starDetails:classF},
	{ starposition: position{x: 4.343384842, y: -1.029772182, z: -1.185420244}, starDetails:class},
	{ starposition: position{x: -16.70166705, y: 3.218436835, z: -4.46001704}, starDetails:classK},
	{ starposition: position{x: 2.074631048, y: -12.15476892, z: -3.186413205}, starDetails:classM},
	{ starposition: position{x: 16.24873665, y: 10.31146801, z: -4.92617235}, starDetails:class},
	{ starposition: position{x: 4.364970878, y: -1.308116204, z: -1.158292685}, starDetails:classM},
	{ starposition: position{x: 0.228781283, y: 14.58496734, z: -3.682361232}, starDetails:classF},
	{ starposition: position{x: 12.4839018, y: -11.72632359, z: -4.245889967}, starDetails:classK},
	{ starposition: position{x: -7.583826129, y: 14.29733823, z: -4.004342185}, starDetails:classG},
	{ starposition: position{x: -8.160685367, y: 6.187753623, z: -2.45661775}, starDetails:classM},
	{ starposition: position{x: 2.805963671, y: -15.53418462, z: -3.754242874}, starDetails:classM},
	{ starposition: position{x: 8.690029729, y: -8.013081561, z: -2.793755766}, starDetails:classM},
	{ starposition: position{x: 2.720463104, y: 3.302911803, z: -1.009795931}, starDetails:class},
	{ starposition: position{x: 7.388367908, y: 6.921868857, z: -2.294458392}, starDetails:classK},
	{ starposition: position{x: -1.587105745, y: -3.845934402, z: -0.9345453215}, starDetails:classM},
	{ starposition: position{x: -6.971451442, y: 10.1060396, z: -2.751010009}, starDetails:classK},
	{ starposition: position{x: 9.035050885, y: -16.07917161, z: -4.11142589}, starDetails:classK},
	{ starposition: position{x: -4.667862219, y: -3.722429726, z: -1.325897225}, starDetails:classM},
	{ starposition: position{x: 13.98721886, y: -10.72930152, z: -3.911166653}, starDetails:classK},
	{ starposition: position{x: -11.30740536, y: 7.124143895, z: -2.920370429}, starDetails:classM},
	{ starposition: position{x: -8.606205136, y: -6.665042018, z: -2.372407541}, starDetails:classM},
	{ starposition: position{x: -12.09944068, y: 5.851054781, z: -2.847012776}, starDetails:classM},
	{ starposition: position{x: 6.91208982, y: -16.2622734, z: -3.522847068}, starDetails:classK},
	{ starposition: position{x: -14.39049297, y: 10.80188005, z: -3.557827139}, starDetails:classK},
	{ starposition: position{x: -13.49207122, y: -13.57755127, z: -3.767139736}, starDetails:classK},
	{ starposition: position{x: 3.822800585, y: 10.1838875, z: -2.12198737}, starDetails:classM},
	{ starposition: position{x: -8.368529795, y: -12.81455389, z: -2.946082459}, starDetails:classK},
	{ starposition: position{x: 15.39826694, y: 8.170676154, z: -3.325968684}, starDetails:classM},
	{ starposition: position{x: 18.87639128, y: -3.067695975, z: -3.635957078}, starDetails:classK},
	{ starposition: position{x: 14.83040253, y: 3.296562328, z: -2.855189349}, starDetails:classF},
	{ starposition: position{x: -4.190850076, y: 14.54423017, z: -2.807153436}, starDetails:classM},
	{ starposition: position{x: 10.1246504, y: -10.41349822, z: -2.677675921}, starDetails:classM},
	{ starposition: position{x: -12.69262749, y: -0.04095261614, z: -2.339814467}, starDetails:classK},
	{ starposition: position{x: -11.25374449, y: 4.94078241, z: -2.21784245}, starDetails:classM},
	{ starposition: position{x: 6.290544275, y: -4.770515166, z: -1.362335008}, starDetails:classM},
	{ starposition: position{x: 5.007814903, y: 7.372184402, z: -1.533830108}, starDetails:classK},
	{ starposition: position{x: 1.899907635, y: 2.542904308, z: -0.5288168683}, starDetails:classK},
	{ starposition: position{x: -12.05692702, y: -15.45431669, z: -3.226390476}, starDetails:classK},
	{ starposition: position{x: -10.44248651, y: 9.844044764, z: -2.223533838}, starDetails:classM},
	{ starposition: position{x: -12.92271725, y: -9.988511442, z: -2.483476228}, starDetails:classK},
	{ starposition: position{x: -16.45335638, y: -6.835248099, z: -2.686153516}, starDetails:classD},
	{ starposition: position{x: -3.76379398, y: -18.36780744, z: -2.775293028}, starDetails:classM},
	{ starposition: position{x: -6.103534083, y: -12.46146279, z: -2.041158146}, starDetails:classG},
	{ starposition: position{x: -17.92854659, y: -7.862890977, z: -2.870564729}, starDetails:classK},
	{ starposition: position{x: -1.577530153, y: -5.455420137, z: -0.8319288354}, starDetails:classM},
	{ starposition: position{x: -1.786875916, y: -6.173425582, z: -0.9399097764}, starDetails:classM},
	{ starposition: position{x: -6.893675332, y: -11.92669639, z: -1.998147473}, starDetails:classM},
	{ starposition: position{x: -13.70678795, y: 1.761924986, z: -1.965806697}, starDetails:class},
	{ starposition: position{x: -1.81488651, y: -9.649070012, z: -1.358351568}, starDetails:class},
	{ starposition: position{x: -4.004189713, y: -4.749198964, z: -0.842322899}, starDetails:classM},
	{ starposition: position{x: 3.83172038, y: -10.44486706, z: -1.497565512}, starDetails:classD},
	{ starposition: position{x: 2.205494653, y: 4.486799965, z: -0.6710412148}, starDetails:classK},
	{ starposition: position{x: -13.47546744, y: 10.44413592, z: -2.204995566}, starDetails:classM},
	{ starposition: position{x: 13.25557251, y: -4.158822846, z: -1.728048061}, starDetails:classK},
	{ starposition: position{x: -16.23365805, y: -10.69587658, z: -2.372982861}, starDetails:classM},
	{ starposition: position{x: -15.68062572, y: 5.73184805, z: -2.027311481}, starDetails:classM},
	{ starposition: position{x: 12.67621088, y: -9.967971901, z: -1.935266836}, starDetails:classM},
	{ starposition: position{x: 7.96006308, y: 13.06516853, z: -1.831695225}, starDetails:classM},
	{ starposition: position{x: -5.018751452, y: 17.87955299, z: -2.213902168}, starDetails:classK},
	{ starposition: position{x: 9.794900305, y: -9.627135212, z: -1.52053038}, starDetails:classM},
	{ starposition: position{x: 17.1482292, y: -0.3241998876, z: -1.845547254}, starDetails:class},
	{ starposition: position{x: 2.239667508, y: 8.47849193, z: -0.8831752723}, starDetails:classK},
	{ starposition: position{x: -14.07744958, y: -12.13626247, z: -1.841247549}, starDetails:classF},
	{ starposition: position{x: -11.7301721, y: 12.32832092, z: -1.618946049}, starDetails:classG},
	{ starposition: position{x: 9.420895266, y: 11.73795763, z: -1.412638755}, starDetails:classK},
	{ starposition: position{x: -1.960419937, y: 8.44005772, z: -0.7845443407}, starDetails:classK},
	{ starposition: position{x: -2.628099642, y: -10.78880035, z: -0.9893676377}, starDetails:classM},
	{ starposition: position{x: -2.543360143, y: -10.40673997, z: -0.9492789975}, starDetails:classK},
	{ starposition: position{x: 12.28211121, y: -12.62984211, z: -1.49373224}, starDetails:classM},
	{ starposition: position{x: -11.96655881, y: -14.52579333, z: -1.572902325}, starDetails:classK},
	{ starposition: position{x: 7.751355023, y: -4.049668226, z: -0.7098941396}, starDetails:classM},
	{ starposition: position{x: -13.13463616, y: -6.133338595, z: -1.061289454}, starDetails:classM},
	{ starposition: position{x: 0.7731342346, y: 15.50522155, z: -1.111198622}, starDetails:classG},
	{ starposition: position{x: 16.35851586, y: -4.241626886, z: -1.136751627}, starDetails:classK},
	{ starposition: position{x: -6.951231284, y: 3.530529668, z: -0.5103312773}, starDetails:classM},
	{ starposition: position{x: -13.30578569, y: 8.218032499, z: -1.008315292}, starDetails:classM},
	{ starposition: position{x: 0.7056646276, y: 5.635176241, z: -0.3644838534}, starDetails:classM},
	{ starposition: position{x: 2.897916536, y: -13.79471018, z: -0.8965075451}, starDetails:classK},
	{ starposition: position{x: -5.987207841, y: 12.84134575, z: -0.8906208006}, starDetails:classK},
	{ starposition: position{x: 1.77885145, y: 12.83509237, z: -0.7919906504}, starDetails:classK},
	{ starposition: position{x: -4.733537751, y: 11.60597743, z: -0.721886788}, starDetails:class},
	{ starposition: position{x: -5.821125153, y: 13.05642562, z: -0.7767492971}, starDetails:class},
	{ starposition: position{x: 2.982157446, y: 16.57585645, z: -0.9045056372}, starDetails:classK},
	{ starposition: position{x: 0.173919896, y: -7.782602633, z: -0.4121276642}, starDetails:classM},
	{ starposition: position{x: 1.756353035, y: -18.82987133, z: -0.9570682402}, starDetails:classK},
	{ starposition: position{x: -0.5257674956, y: 4.078380706, z: -0.202014267}, starDetails:classM},
	{ starposition: position{x: -13.51161602, y: 10.4479262, z: -0.8260703523}, starDetails:classF},
	{ starposition: position{x: -8.780510614, y: -5.123783349, z: -0.4716561967}, starDetails:classK},
	{ starposition: position{x: 18.99210573, y: -5.24386338, z: -0.8241865339}, starDetails:classG},
	{ starposition: position{x: -3.137376527, y: -19.18733245, z: -0.8107590339}, starDetails:classG},
	{ starposition: position{x: -13.10996344, y: -5.320062018, z: -0.5828722876}, starDetails:classM},
	{ starposition: position{x: -3.487198627, y: -9.126461031, z: -0.3964747158}, starDetails:classK},
	{ starposition: position{x: 8.894313769, y: 13.61620595, z: -0.5598551801}, starDetails:class},
	{ starposition: position{x: 1.668456474, y: -19.21795629, z: -0.6529376709}, starDetails:classK},
	{ starposition: position{x: -2.250281822, y: -10.55870458, z: -0.3488867489}, starDetails:classG},
	{ starposition: position{x: 2.437262856, y: -18.59322409, z: -0.5950909755}, starDetails:classK},
	{ starposition: position{x: -11.63142273, y: -2.138265017, z: -0.2992575212}, starDetails:classF},
	{ starposition: position{x: -9.213717248, y: 16.86900101, z: -0.473820934}, starDetails:classG},
	{ starposition: position{x: -11.46870932, y: -12.9471402, z: -0.4076580437}, starDetails:classK},
	{ starposition: position{x: 8.033233036, y: 13.567254, z: -0.3190186619}, starDetails:classK},
	{ starposition: position{x: -2.118136686, y: -16.3075568, z: -0.304985595}, starDetails:classG},
	{ starposition: position{x: 10.97805001, y: -16.58633321, z: -0.301181704}, starDetails:classK},
	{ starposition: position{x: 16.36976377, y: -6.270585031, z: -0.257063035}, starDetails:classK},
	{ starposition: position{x: -8.596567843, y: 15.19985613, z: -0.2482346208}, starDetails:classK},
	{ starposition: position{x: -10.51771041, y: -2.3667196, z: -0.14435739}, starDetails:classM},
	{ starposition: position{x: 9.424928594, y: 16.76261007, z: -0.09005547839}, starDetails:classF},
	{ starposition: position{x: -4.88229529, y: -16.21874814, z: -0.006744538558}, starDetails:classG},
	{ starposition: position{x: 14.61715328, y: -12.47926326, z: 0.05435352259}, starDetails:classK},
	{ starposition: position{x: 13.88483845, y: -1.577974438, z: 0.04431331594}, starDetails:classM},
	{ starposition: position{x: 8.021356216, y: 11.12959195, z: 0.0964564008}, starDetails:classF},
	{ starposition: position{x: 17.16726359, y: 7.868304435, z: 0.2146373517}, starDetails:classK},
	{ starposition: position{x: -10.22039307, y: -11.75559923, z: 0.2166145773}, starDetails:classK},
	{ starposition: position{x: -3.332895902, y: 0.1785480904, z: 0.04704427485}, starDetails:classM},
	{ starposition: position{x: -14.26854764, y: -10.29577739, z: 0.3816969534}, starDetails:classG},
	{ starposition: position{x: -5.122609432, y: 7.584819521, z: 0.2080901031}, starDetails:class},
	{ starposition: position{x: 8.975794855, y: -5.069183025, z: 0.2520935559}, starDetails:classM},
	{ starposition: position{x: 19.18275657, y: -2.100744704, z: 0.5408256833}, starDetails:classM},
	{ starposition: position{x: -10.88620484, y: 0.4425589847, z: 0.3358051786}, starDetails:classF},
	{ starposition: position{x: 1.921374378, y: 16.70099966, z: 0.5711307116}, starDetails:classM},
	{ starposition: position{x: 10.14256066, y: 10.72937926, z: 0.5071837185}, starDetails:classM},
	{ starposition: position{x: -1.148261406, y: -7.628431529, z: 0.2848012892}, starDetails:classK},
	{ starposition: position{x: -6.241728362, y: 13.47968646, z: 0.5665725445}, starDetails:classM},
	{ starposition: position{x: 5.957918217, y: -0.2809918093, z: 0.2503612964}, starDetails:classM},
	{ starposition: position{x: 0.1208964155, y: -5.079663981, z: 0.2220624569}, starDetails:classK},
	{ starposition: position{x: -8.194584209, y: -12.15248927, z: 0.6439273955}, starDetails:classG},
	{ starposition: position{x: 3.228358111, y: 19.62008379, z: 0.9041505629}, starDetails:classK},
	{ starposition: position{x: -14.642793, y: -13.10035036, z: 0.927734445}, starDetails:classG},
	{ starposition: position{x: -14.95062022, y: -5.713007047, z: 0.7611709241}, starDetails:classG},
	{ starposition: position{x: -15.6652576, y: -5.988301087, z: 0.7981623972}, starDetails:classG},
	{ starposition: position{x: 9.447747906, y: 14.54438495, z: 0.84485965}, starDetails:classM},
	{ starposition: position{x: 3.14083392, y: -9.634174703, z: 0.5111402476}, starDetails:classM},
	{ starposition: position{x: -17.77569307, y: 2.594904752, z: 0.943301918}, starDetails:classK},
	{ starposition: position{x: -17.46129292, y: 2.550250621, z: 0.9287260808}, starDetails:classK},
	{ starposition: position{x: 5.594395188, y: -14.29432686, z: 0.8352481852}, starDetails:classF},
	{ starposition: position{x: -9.796290679, y: -10.65329873, z: 0.800447999}, starDetails:classM},
	{ starposition: position{x: 6.68275293, y: -11.25671951, z: 0.7607556021}, starDetails:classK},
	{ starposition: position{x: 15.23626297, y: -3.509778964, z: 0.9092922784}, starDetails:classM},
	{ starposition: position{x: 5.896798599, y: 6.987745382, z: 0.5384060744}, starDetails:classG},
	{ starposition: position{x: -2.610833271, y: 5.313348349, z: 0.3676456691}, starDetails:classM},
	{ starposition: position{x: -0.9650299316, y: -10.6469634, z: 0.6642563361}, starDetails:classK},
	{ starposition: position{x: 8.684695036, y: 5.659697456, z: 0.648569634}, starDetails:classM},
	{ starposition: position{x: 10.10005186, y: 4.890193427, z: 0.848390511}, starDetails:classM},
	{ starposition: position{x: 13.23258984, y: 9.950080717, z: 1.283101626}, starDetails:classK},
	{ starposition: position{x: -13.81162188, y: 12.1236225, z: 1.427933154}, starDetails:classK},
	{ starposition: position{x: -17.52061331, y: 3.626458946, z: 1.402607413}, starDetails:classK},
	{ starposition: position{x: 5.791014743, y: -13.21440151, z: 1.15619569}, starDetails:classK},
	{ starposition: position{x: -0.01729902865, y: -1.815335484, z: 0.1482429847}, starDetails:classs},
	{ starposition: position{x: 11.8748242, y: -14.19960318, z: 1.610300237}, starDetails:classK},
	{ starposition: position{x: -1.448778937, y: 19.18884645, z: 1.717330144}, starDetails:classG},
	{ starposition: position{x: 1.926742094, y: -5.523024391, z: 0.5294745278}, starDetails:classM},
	{ starposition: position{x: -1.462376302, y: 3.160924416, z: 0.318646848}, starDetails:classF},
	{ starposition: position{x: -1.40787335, y: 3.510923825, z: 0.3465654417}, starDetails:classM},
	{ starposition: position{x: 7.263886329, y: 1.556431813, z: 0.6869733044}, starDetails:classK},
	{ starposition: position{x: 4.485730647, y: 10.85746122, z: 1.089487768}, starDetails:classF},
	{ starposition: position{x: 4.286233238, y: 0.9336282769, z: 0.4142977288}, starDetails:classD},
	{ starposition: position{x: 14.37219869, y: 5.081464496, z: 1.461249289}, starDetails:classM},
	{ starposition: position{x: -1.280668368, y: -9.853367567, z: 0.9653502731}, starDetails:classM},
	{ starposition: position{x: 13.67223353, y: -1.199351173, z: 1.352337925}, starDetails:classF},
	{ starposition: position{x: -13.21216663, y: -13.78932303, z: 1.885742775}, starDetails:classK},
	{ starposition: position{x: -10.00732149, y: 7.684531896, z: 1.249345688}, starDetails:classK},
	{ starposition: position{x: -1.450438612, y: -19.53084736, z: 1.985699941}, starDetails:classM},
	{ starposition: position{x: 11.66643166, y: 12.76060499, z: 1.789815229}, starDetails:classM},
	{ starposition: position{x: 2.766788719, y: -10.6975711, z: 1.144024234}, starDetails:classM},
	{ starposition: position{x: 6.568503634, y: -11.93415026, z: 1.529904471}, starDetails:classG},
	{ starposition: position{x: 3.515699621, y: 11.5162315, z: 1.367055786}, starDetails:classM},
	{ starposition: position{x: -5.342736061, y: 1.662143329, z: 0.6682029489}, starDetails:classM},
	{ starposition: position{x: 5.560627695, y: 4.505474144, z: 0.8639604074}, starDetails:classK},
	{ starposition: position{x: 2.401026949, y: 7.596084873, z: 0.9726966075}, starDetails:classF},
	{ starposition: position{x: -15.7976624, y: 2.537771349, z: 1.971722506}, starDetails:classM},
	{ starposition: position{x: 10.653797, y: -10.16958237, z: 1.826854343}, starDetails:classK},
	{ starposition: position{x: 16.88896266, y: -10.00466651, z: 2.806658763}, starDetails:classM},
	{ starposition: position{x: -6.871422821, y: -16.17841842, z: 2.253588312}, starDetails:classK},
	{ starposition: position{x: -6.415000096, y: -9.733086878, z: 1.504304542}, starDetails:classG},
	{ starposition: position{x: -16.74083563, y: 4.759605672, z: 2.256920996}, starDetails:classK},
	{ starposition: position{x: 16.90648008, y: -0.107457416, z: 2.273695609}, starDetails:classM},
	{ starposition: position{x: -2.77967409, y: -14.53220072, z: 2.094815521}, starDetails:classM},
	{ starposition: position{x: 2.747718199, y: -11.13301873, z: 1.693859161}, starDetails:classM},
	{ starposition: position{x: -13.30236321, y: -1.690268441, z: 1.986369069}, starDetails:classM},
	{ starposition: position{x: 12.25874062, y: -6.20590076, z: 2.066920499}, starDetails:classM},
	{ starposition: position{x: -16.76015164, y: -7.247989083, z: 2.757210752}, starDetails:classK},
	{ starposition: position{x: -13.12701655, y: -1.802243482, z: 2.054043548}, starDetails:classM},
	{ starposition: position{x: 2.361665817, y: -4.499358237, z: 0.7927791313}, starDetails:classA},
	{ starposition: position{x: -11.42904043, y: -8.374575068, z: 2.215549514}, starDetails:classM},
	{ starposition: position{x: 11.73879227, y: -4.681415352, z: 2.087112069}, starDetails:classM},
	{ starposition: position{x: 12.06712308, y: -10.44000373, z: 2.639893711}, starDetails:classK},
	{ starposition: position{x: -16.72342357, y: -5.821957789, z: 2.939039965}, starDetails:classG},
	{ starposition: position{x: -9.96398861, y: 11.5772275, z: 2.571055705}, starDetails:classM},
	{ starposition: position{x: -8.577051148, y: 9.977018796, z: 2.215477738}, starDetails:classM},
	{ starposition: position{x: 1.192512982, y: 8.486359452, z: 1.456505719}, starDetails:classM},
	{ starposition: position{x: -12.69109703, y: -10.27771883, z: 2.805303294}, starDetails:classK},
	{ starposition: position{x: -7.916352437, y: -1.681000812, z: 1.390974}, starDetails:classM},
	{ starposition: position{x: 1.520220336, y: 12.49205849, z: 2.17849961}, starDetails:classM},
	{ starposition: position{x: -17.45550444, y: -2.687921345, z: 3.059917739}, starDetails:classM},
	{ starposition: position{x: 13.65596857, y: -12.03090396, z: 3.211621578}, starDetails:classF},
	{ starposition: position{x: 9.227429021, y: -4.999818426, z: 1.867793156}, starDetails:classA},
	{ starposition: position{x: 15.90820457, y: 1.271315016, z: 2.872286097}, starDetails:classM},
	{ starposition: position{x: -17.42441334, y: 4.206325038, z: 3.237368076}, starDetails:classK},
	{ starposition: position{x: -0.5047747638, y: 10.5929339, z: 1.931131964}, starDetails:classM},
	{ starposition: position{x: -6.930937597, y: -2.870366991, z: 1.374099385}, starDetails:classM},
	{ starposition: position{x: 8.881841152, y: -16.87713454, z: 3.505812069}, starDetails:classF},
	{ starposition: position{x: -1.025527741, y: 17.77561867, z: 3.341111971}, starDetails:classG},
	{ starposition: position{x: 3.569855034, y: -16.62151665, z: 3.226582878}, starDetails:classK},
	{ starposition: position{x: -13.99463647, y: -8.370504791, z: 3.107027164}, starDetails:classG},
	{ starposition: position{x: -3.389132271, y: -17.44044032, z: 3.472328042}, starDetails:classM},
	{ starposition: position{x: 1.141001851, y: 11.09642662, z: 2.234646673}, starDetails:classK},
	{ starposition: position{x: -11.65859123, y: 13.9129446, z: 3.700934841}, starDetails:classK},
	{ starposition: position{x: -11.20939844, y: 11.49889082, z: 3.310285841}, starDetails:classM},
	{ starposition: position{x: -14.83968336, y: -2.54229142, z: 3.117034626}, starDetails:classM},
	{ starposition: position{x: 11.65661217, y: 10.33879903, z: 3.248157666}, starDetails:classK},
	{ starposition: position{x: -5.284268171, y: -17.89843654, z: 3.937474734}, starDetails:classM},
	{ starposition: position{x: 5.369565614, y: -13.81444585, z: 3.134918187}, starDetails:classG},
	{ starposition: position{x: 15.0784484, y: -4.99469949, z: 3.426757675}, starDetails:classF},
	{ starposition: position{x: -1.373449647, y: 19.11052596, z: 4.167606777}, starDetails:classF},
	{ starposition: position{x: -10.77270803, y: -2.925998005, z: 2.44939629}, starDetails:classM},
	{ starposition: position{x: 0.4400389595, y: 5.633504946, z: 1.252013986}, starDetails:classM},
	{ starposition: position{x: -1.525532953, y: -13.8922438, z: 3.113855665}, starDetails:classA},
	{ starposition: position{x: -5.93582962, y: -14.83327678, z: 3.575089795}, starDetails:classK},
	{ starposition: position{x: -3.356390734, y: 16.76262081, z: 3.914117892}, starDetails:classF},
	{ starposition: position{x: -9.900546635, y: 14.87295518, z: 4.132316008}, starDetails:classK},
	{ starposition: position{x: -8.932792468, y: -14.46386479, z: 3.986672355}, starDetails:classG},
	{ starposition: position{x: -9.08856993, y: 6.294440704, z: 2.595040421}, starDetails:classM},
	{ starposition: position{x: -16.33369, y: -9.652608564, z: 4.621523277}, starDetails:classM},
	{ starposition: position{x: -16.29497007, y: -6.619393678, z: 4.313611665}, starDetails:classG},
	{ starposition: position{x: -14.23548851, y: -12.52880155, z: 4.674368955}, starDetails:classK},
	{ starposition: position{x: 1.290922186, y: -18.41886881, z: 4.57438702}, starDetails:classM},
	{ starposition: position{x: -10.72237412, y: 0.5120014747, z: 2.790617864}, starDetails:classA},
	{ starposition: position{x: 14.76659761, y: -9.911537599, z: 4.689661124}, starDetails:classG},
	{ starposition: position{x: -4.699965887, y: -2.335939661, z: 1.396021447}, starDetails:classM},
	{ starposition: position{x: -11.60664105, y: 14.18888924, z: 4.956989135}, starDetails:classK},
	{ starposition: position{x: 14.26548057, y: 6.647032757, z: 4.2934789}, starDetails:classK},
	{ starposition: position{x: -12.57083149, y: 12.4298315, z: 4.825173549}, starDetails:classM},
	{ starposition: position{x: -13.56493107, y: -2.45946835, z: 3.791516563}, starDetails:classK},
	{ starposition: position{x: -10.9285435, y: -8.348372039, z: 3.82149826}, starDetails:classM},
	{ starposition: position{x: -0.433623411, y: 14.88214998, z: 4.140792524}, starDetails:classK},
	{ starposition: position{x: 2.22640806, y: 16.60387625, z: 4.670317717}, starDetails:classM},
	{ starposition: position{x: 9.493776836, y: -16.02581691, z: 5.198421613}, starDetails:classG},
	{ starposition: position{x: -5.496969012, y: -9.189306214, z: 3.002756208}, starDetails:classF},
	{ starposition: position{x: -6.824846971, y: -6.504013379, z: 2.721395723}, starDetails:classM},
	{ starposition: position{x: 13.69004696, y: -7.760173, z: 4.651531987}, starDetails:classM},
	{ starposition: position{x: -5.027303745, y: -15.07586676, z: 4.701132485}, starDetails:classM},
	{ starposition: position{x: -12.60836409, y: -11.19316102, z: 4.993803264}, starDetails:classK},
	{ starposition: position{x: -15.65627883, y: 4.025530417, z: 4.791098783}, starDetails:classM},
	{ starposition: position{x: -16.15988003, y: 4.156612664, z: 4.945661257}, starDetails:class},
	{ starposition: position{x: 6.865747864, y: 17.86724237, z: 5.673361895}, starDetails:classK},
	{ starposition: position{x: 6.34746443, y: -1.802417854, z: 1.9613198}, starDetails:classM},
	{ starposition: position{x: 3.352622175, y: -14.56373325, z: 4.450777012}, starDetails:classM},
	{ starposition: position{x: 8.738670715, y: 12.92548512, z: 4.671431502}, starDetails:classM},
	{ starposition: position{x: 9.229104913, y: 13.6665573, z: 4.938986771}, starDetails:classM},
	{ starposition: position{x: -12.11052936, y: -5.186865032, z: 3.980485047}, starDetails:classM},
	{ starposition: position{x: -10.13769732, y: -3.532524707, z: 3.285832024}, starDetails:classK},
	{ starposition: position{x: 8.705497351, y: -14.472963, z: 5.18655838}, starDetails:classG},
	{ starposition: position{x: 10.06890825, y: -1.701690442, z: 3.179365475}, starDetails:classM},
	{ starposition: position{x: 2.16213725, y: 13.51268172, z: 4.268591684}, starDetails:classK},
	{ starposition: position{x: 2.163870381, y: 13.82680723, z: 4.381387965}, starDetails:classF},
	{ starposition: position{x: 3.445221466, y: -15.91169016, z: 5.114445088}, starDetails:classM},
	{ starposition: position{x: -13.27745132, y: -6.711816171, z: 4.678512189}, starDetails:classF},
	{ starposition: position{x: -13.0273537, y: -4.107039554, z: 4.314430806}, starDetails:classF},
	{ starposition: position{x: -1.516091732, y: 9.261317083, z: 2.97047778}, starDetails:classM},
	{ starposition: position{x: 5.091604154, y: -3.920947629, z: 2.043797886}, starDetails:classM},
	{ starposition: position{x: -11.13220414, y: -5.494631823, z: 3.983407163}, starDetails:class},
	{ starposition: position{x: -6.094924506, y: 17.08169456, z: 5.856141678}, starDetails:classA},
	{ starposition: position{x: -11.67265628, y: 12.54599801, z: 5.609320052}, starDetails:class},
	{ starposition: position{x: -9.442415655, y: -5.163429502, z: 3.579739629}, starDetails:classG},
	{ starposition: position{x: -7.175115395, y: -17.08589017, z: 6.169262734}, starDetails:classK},
	{ starposition: position{x: 9.658923672, y: -4.948136203, z: 3.615597914}, starDetails:classM},
	{ starposition: position{x: -0.7424879202, y: -7.668804991, z: 2.591307304}, starDetails:classM},
	{ starposition: position{x: 3.417673705, y: 14.64159808, z: 5.073051476}, starDetails:classG},
	{ starposition: position{x: -1.582069309, y: 13.79357297, z: 4.714958388}, starDetails:classK},
	{ starposition: position{x: 2.939925255, y: 8.409554588, z: 3.060686861}, starDetails:classM},
	{ starposition: position{x: -4.641615229, y: -4.305263338, z: 2.192337823}, starDetails:classG},
	{ starposition: position{x: 17.48986055, y: -1.255334676, z: 6.082244205}, starDetails:classG},
	{ starposition: position{x: -7.923901616, y: -7.480216788, z: 3.784517793}, starDetails:classK},
	{ starposition: position{x: -8.820912124, y: -5.931455785, z: 3.699007498}, starDetails:classK},
	{ starposition: position{x: -9.066339886, y: 16.5428407, z: 6.583092059}, starDetails:classK},
	{ starposition: position{x: 5.758130536, y: -17.04619622, z: 6.306740613}, starDetails:classM},
	{ starposition: position{x: -7.42498911, y: 9.535356621, z: 4.254590133}, starDetails:classM},
	{ starposition: position{x: 13.23990484, y: 11.44292304, z: 6.172631984}, starDetails:classK},
	{ starposition: position{x: 2.458186355, y: 11.74130495, z: 4.286894748}, starDetails:classM},
	{ starposition: position{x: 12.3275374, y: -14.13556733, z: 6.734466236}, starDetails:classM},
	{ starposition: position{x: 5.829003996, y: -0.7192252597, z: 2.130410063}, starDetails:classM},
	{ starposition: position{x: 14.75737586, y: 4.177931347, z: 5.611997127}, starDetails:classM},
	{ starposition: position{x: 10.18918019, y: 8.242636793, z: 4.827227147}, starDetails:classM},
	{ starposition: position{x: 6.31622862, y: 3.029590666, z: 2.587163}, starDetails:classK},
	{ starposition: position{x: 0.1990871368, y: 8.123972326, z: 3.002248576}, starDetails:classG},
	{ starposition: position{x: -6.029667422, y: 11.80102099, z: 4.920695867}, starDetails:classM},
	{ starposition: position{x: -16.23869692, y: 3.295915432, z: 6.203118471}, starDetails:classA},
	{ starposition: position{x: 3.538900092, y: -17.52644325, z: 6.701883508}, starDetails:classF},
	{ starposition: position{x: -16.66887236, y: -4.95789541, z: 6.581409756}, starDetails:classM},
	{ starposition: position{x: 13.83143977, y: -3.87064486, z: 5.44694264}, starDetails:classG},
	{ starposition: position{x: -2.035134348, y: 16.71607949, z: 6.391090147}, starDetails:classG},
	{ starposition: position{x: 14.9844443, y: 8.190079996, z: 6.48963046}, starDetails:classA},
	{ starposition: position{x: 2.394544684, y: -7.927980096, z: 3.158805339}, starDetails:classM},
	{ starposition: position{x: 2.326249905, y: -7.717356483, z: 3.076152329}, starDetails:classM},
	{ starposition: position{x: 4.242055531, y: 11.86663603, z: 4.812798367}, starDetails:classK},
	{ starposition: position{x: -6.769481425, y: -8.203381576, z: 4.077917292}, starDetails:classM},
	{ starposition: position{x: -8.475512363, y: 13.53435019, z: 6.163871528}, starDetails:classM},
	{ starposition: position{x: 10.19974063, y: 1.769513177, z: 4.025995633}, starDetails:classK},
	{ starposition: position{x: -2.876121767, y: -12.24550289, z: 4.968613804}, starDetails:classM},
	{ starposition: position{x: 4.115383985, y: 9.814409005, z: 4.282992698}, starDetails:classK},
	{ starposition: position{x: -10.47691681, y: 2.678443423, z: 4.361902898}, starDetails:classK},
	{ starposition: position{x: -5.728101631, y: 15.71155516, z: 6.750600265}, starDetails:classF},
	{ starposition: position{x: 7.438442527, y: 13.60572984, z: 6.267867787}, starDetails:classG},
	{ starposition: position{x: -6.779275319, y: 16.88960863, z: 7.369371528}, starDetails:classM},
	{ starposition: position{x: 8.827237562, y: -9.028685032, z: 5.19493699}, starDetails:classM},
	{ starposition: position{x: -14.82753004, y: 1.838366873, z: 6.239890059}, starDetails:class},
	{ starposition: position{x: 8.928913135, y: -15.35241134, z: 7.433396079}, starDetails:classG},
	{ starposition: position{x: -5.897654795, y: 1.578191001, z: 2.570593654}, starDetails:classM},
	{ starposition: position{x: 7.362755525, y: -12.26829977, z: 6.174741221}, starDetails:classK},
	{ starposition: position{x: -6.800599769, y: -6.445923161, z: 4.085019651}, starDetails:class},
	{ starposition: position{x: -12.38138418, y: -9.138034732, z: 6.729351467}, starDetails:classM},
	{ starposition: position{x: -11.63445581, y: -8.590919417, z: 6.32549272}, starDetails:classM},
	{ starposition: position{x: -10.62873818, y: -5.593473342, z: 5.288246153}, starDetails:class},
	{ starposition: position{x: 11.38780618, y: 14.07603716, z: 7.980039324}, starDetails:classM},
	{ starposition: position{x: -12.08130169, y: -11.09590611, z: 7.273090173}, starDetails:classG},
	{ starposition: position{x: -2.867194533, y: 15.75383811, z: 7.115350579}, starDetails:classK},
	{ starposition: position{x: -0.5929681987, y: -4.444555432, z: 2.05794658}, starDetails:class},
	{ starposition: position{x: 10.64499513, y: 2.386848277, z: 5.044694478}, starDetails:class},
	{ starposition: position{x: -15.2326379, y: -3.292558856, z: 7.214380721}, starDetails:classG},
	{ starposition: position{x: -12.24420388, y: -13.0500584, z: 8.294958241}, starDetails:classF},
	{ starposition: position{x: 8.352269668, y: -10.46560383, z: 6.262367494}, starDetails:classD},
	{ starposition: position{x: 9.359469665, y: -5.028608215, z: 5.032547889}, starDetails:classF},
	{ starposition: position{x: -3.730282965, y: 15.15515549, z: 7.40284979}, starDetails:classG},
	{ starposition: position{x: 5.138257863, y: 4.476436766, z: 3.253987728}, starDetails:classM},
	{ starposition: position{x: -2.478298366, y: -8.959029273, z: 4.482692013}, starDetails:classM},
	{ starposition: position{x: 7.200226656, y: 10.91261571, z: 6.43771682}, starDetails:classK},
	{ starposition: position{x: -1.87647218, y: -10.61648416, z: 5.375229628}, starDetails:classM},
	{ starposition: position{x: -9.112901772, y: 0.7101939136, z: 4.598918156}, starDetails:classM},
	{ starposition: position{x: -10.46514385, y: 7.063266996, z: 6.424595988}, starDetails:classM},
	{ starposition: position{x: -10.81410364, y: -5.5738843, z: 6.193603589}, starDetails:classK},
	{ starposition: position{x: -12.65953744, y: 9.47649581, z: 8.053619066}, starDetails:classK},
	{ starposition: position{x: 11.04175277, y: 0.1041053369, z: 5.646829898}, starDetails:classG},
	{ starposition: position{x: 5.641259355, y: 14.8510137, z: 8.140853051}, starDetails:classK},
	{ starposition: position{x: -3.589450161, y: 10.37595776, z: 5.628778276}, starDetails:class},
	{ starposition: position{x: -9.249014844, y: 13.2010485, z: 8.290501489}, starDetails:classF},
	{ starposition: position{x: -0.4389724605, y: -7.422654879, z: 3.907515248}, starDetails:classG},
	{ starposition: position{x: 9.473607891, y: -6.757382314, z: 6.115574194}, starDetails:class},
	{ starposition: position{x: 15.66756527, y: -7.766721827, z: 9.240873432}, starDetails:classM},
	{ starposition: position{x: -7.697863089, y: -2.496807884, z: 4.280510517}, starDetails:classG},
	{ starposition: position{x: -4.047310397, y: 8.178081728, z: 4.857111476}, starDetails:classK},
	{ starposition: position{x: 6.85479263, y: -3.909324816, z: 4.250183479}, starDetails:class},
	{ starposition: position{x: -7.543814212, y: 8.047251944, z: 5.946998635}, starDetails:classG},
	{ starposition: position{x: -11.70190084, y: 10.00405101, z: 8.380466813}, starDetails:classK},
	{ starposition: position{x: 11.97684979, y: 0.3454663384, z: 6.647635312}, starDetails:classK},
	{ starposition: position{x: -7.361685437, y: 12.68204551, z: 8.200453616}, starDetails:classG},
	{ starposition: position{x: -15.10579626, y: -5.764689301, z: 9.049870828}, starDetails:classK},
	{ starposition: position{x: -4.55366925, y: 16.01072546, z: 9.356069484}, starDetails:classG},
	{ starposition: position{x: 15.05680589, y: 8.097229063, z: 9.703738489}, starDetails:classF},
	{ starposition: position{x: -10.19361535, y: -7.268396141, z: 7.120878616}, starDetails:classK},
	{ starposition: position{x: -10.48395171, y: -8.390083275, z: 7.673025329}, starDetails:classF},
	{ starposition: position{x: -13.22665445, y: 1.326798298, z: 7.612855276}, starDetails:classD},
	{ starposition: position{x: 7.076398309, y: -11.82195479, z: 7.922102339}, starDetails:classG},
	{ starposition: position{x: -12.47275491, y: -8.252075912, z: 8.710154971}, starDetails:classK},
	{ starposition: position{x: -10.16286234, y: -12.46131902, z: 9.392000332}, starDetails:classG},
	{ starposition: position{x: -10.04814585, y: 2.178015431, z: 6.043333816}, starDetails:classK},
	{ starposition: position{x: 0.4132238739, y: -13.47332375, z: 7.95974414}, starDetails:classF},
	{ starposition: position{x: -3.946539305, y: 15.51762846, z: 9.528574593}, starDetails:classM},
	{ starposition: position{x: 11.56521049, y: -3.542116723, z: 7.486177814}, starDetails:classM},
	{ starposition: position{x: -3.096058345, y: -8.657820003, z: 5.657065271}, starDetails:classF},
	{ starposition: position{x: -5.960862986, y: 14.55032926, z: 9.743241157}, starDetails:classF},
	{ starposition: position{x: 1.756109409, y: -9.478035451, z: 5.983749517}, starDetails:classM},
	{ starposition: position{x: -5.383373817, y: 12.29279258, z: 8.349453874}, starDetails:classA},
	{ starposition: position{x: -10.97571481, y: 6.272057453, z: 7.876146715}, starDetails:classG},
	{ starposition: position{x: -5.637593292, y: 13.7753676, z: 9.298381872}, starDetails:classK},
	{ starposition: position{x: 5.068426341, y: -10.12213133, z: 7.078581318}, starDetails:classM},
	{ starposition: position{x: 4.296156379, y: -8.63329857, z: 6.116593169}, starDetails:class},
	{ starposition: position{x: 12.38407696, y: -5.547315045, z: 8.631549369}, starDetails:classM},
	{ starposition: position{x: -2.074210305, y: -11.96416215, z: 7.7268559}, starDetails:classG},
	{ starposition: position{x: 13.98873489, y: -0.9875195884, z: 8.949980709}, starDetails:classM},
	{ starposition: position{x: -5.647835171, y: 15.61976556, z: 10.71678025}, starDetails:classM},
	{ starposition: position{x: 3.097039694, y: -12.18916291, z: 8.136582398}, starDetails:classG},
	{ starposition: position{x: 4.622185645, y: -12.10971022, z: 8.488971194}, starDetails:classG},
	{ starposition: position{x: -1.092585525, y: 4.479891506, z: 3.02543927}, starDetails:classK},
	{ starposition: position{x: -6.855219154, y: -3.296337665, z: 4.995026924}, starDetails:classM},
	{ starposition: position{x: -7.224957842, y: -12.64702912, z: 9.569561096}, starDetails:classG},
	{ starposition: position{x: -13.65416307, y: -4.947606185, z: 9.556787926}, starDetails:classM},
	{ starposition: position{x: -2.615064437, y: -7.715487544, z: 5.393822587}, starDetails:classK},
	{ starposition: position{x: 5.326465223, y: 10.08303395, z: 7.586979711}, starDetails:classM},
	{ starposition: position{x: -11.66050191, y: 10.79269646, z: 10.66969441}, starDetails:classF},
	{ starposition: position{x: -7.86403954, y: 0.6517093246, z: 5.363231894}, starDetails:classG},
	{ starposition: position{x: 7.411855627, y: 5.048460328, z: 6.10023591}, starDetails:classG},
	{ starposition: position{x: -13.20419892, y: 6.967694295, z: 10.16245929}, starDetails:classK},
	{ starposition: position{x: -12.47824735, y: -4.601105038, z: 9.065392715}, starDetails:classM},
	{ starposition: position{x: 8.59447034, y: 8.021993946, z: 8.047418716}, starDetails:classK},
	{ starposition: position{x: 14.82925601, y: -4.631690524, z: 10.81987354}, starDetails:class},
	{ starposition: position{x: -9.660868984, y: 12.77617841, z: 11.22282614}, starDetails:classM},
	{ starposition: position{x: -13.86524529, y: -5.017358042, z: 10.36649277}, starDetails:classM},
	{ starposition: position{x: -10.12687079, y: -3.663640187, z: 7.57210729}, starDetails:classM},
	{ starposition: position{x: -14.31444257, y: -3.680260876, z: 10.43594705}, starDetails:classM},
	{ starposition: position{x: -6.97753983, y: 0.2703207301, z: 4.938848203}, starDetails:classM},
	{ starposition: position{x: 7.354718656, y: 13.16367784, z: 10.66723649}, starDetails:classK},
	{ starposition: position{x: -8.063311906, y: -3.652406418, z: 6.364969077}, starDetails:classM},
	{ starposition: position{x: -7.326802083, y: 5.339509694, z: 6.541197078}, starDetails:classG},
	{ starposition: position{x: -1.999506206, y: 0.5046205999, z: 1.497256617}, starDetails:classM},
	{ starposition: position{x: -3.596734697, y: 8.477004948, z: 6.744505412}, starDetails:classM},
	{ starposition: position{x: -3.615004152, y: 8.520697095, z: 6.781807417}, starDetails:classM},
	{ starposition: position{x: 15.84976106, y: -0.9818499966, z: 11.64590567}, starDetails:classM},
	{ starposition: position{x: -5.259958537, y: -14.61341594, z: 11.41561193}, starDetails:class},
	{ starposition: position{x: -8.707924768, y: 6.565382725, z: 8.017265879}, starDetails:classM},
	{ starposition: position{x: -6.204048602, y: 14.43852821, z: 11.85493395}, starDetails:classG},
	{ starposition: position{x: -2.521566211, y: 11.95638784, z: 9.382413724}, starDetails:classD},
	{ starposition: position{x: -7.238320484, y: 0.2233549947, z: 5.603699334}, starDetails:classG},
	{ starposition: position{x: 8.825762059, y: 10.34452531, z: 10.72106225}, starDetails:classM},
	{ starposition: position{x: -1.975914681, y: -9.212207909, z: 7.479056556}, starDetails:class},
	{ starposition: position{x: 0.3649417623, y: -8.68231629, z: 6.902156227}, starDetails:classK},
	{ starposition: position{x: 7.672874302, y: -12.62465136, z: 11.74215189}, starDetails:classG},
	{ starposition: position{x: -10.46071007, y: 3.532689914, z: 8.786092597}, starDetails:classM},
	{ starposition: position{x: -1.496745369, y: 4.744469702, z: 3.961768176}, starDetails:classM},
	{ starposition: position{x: 1.989579059, y: -1.873739537, z: 2.192247559}, starDetails:classK},
	{ starposition: position{x: 1.977350225, y: -1.862591052, z: 2.179527363}, starDetails:classK},
	{ starposition: position{x: 0.9702107471, y: -5.967749975, z: 4.858236685}, starDetails:classA},
	{ starposition: position{x: -5.359979989, y: -9.765276222, z: 9.071098428}, starDetails:classG},
	{ starposition: position{x: -3.740033603, y: 10.54120024, z: 9.146718699}, starDetails:classM},
	{ starposition: position{x: -13.18754306, y: -2.622842635, z: 10.99689937}, starDetails:classG},
	{ starposition: position{x: 13.66437424, y: -5.143033275, z: 11.98499339}, starDetails:classM},
	{ starposition: position{x: 8.175569509, y: -8.169771007, z: 9.722621527}, starDetails:classM},
	{ starposition: position{x: 1.715595223, y: 9.519299937, z: 8.145328006}, starDetails:classG},
	{ starposition: position{x: 12.98513038, y: 2.337366625, z: 11.14522212}, starDetails:classK},
	{ starposition: position{x: -10.3435099, y: 2.797175694, z: 9.128944036}, starDetails:classG},
	{ starposition: position{x: 11.29069758, y: 0.8439447519, z: 9.824239938}, starDetails:classM},
	{ starposition: position{x: -5.035222715, y: -10.90711306, z: 10.42869038}, starDetails:classM},
	{ starposition: position{x: -3.02379394, y: 5.333189676, z: 5.387024324}, starDetails:classM},
	{ starposition: position{x: -6.215343092, y: -0.9220018062, z: 5.53110401}, starDetails:classG},
	{ starposition: position{x: 9.213800504, y: 4.140827602, z: 8.907682853}, starDetails:classF},
	{ starposition: position{x: -1.605604833, y: -9.076849158, z: 8.217294428}, starDetails:classM},
	{ starposition: position{x: -8.687990783, y: 8.639160279, z: 10.94835741}, starDetails:classF},
	{ starposition: position{x: -11.80335391, y: -1.131281036, z: 10.72723982}, starDetails:classM},
	{ starposition: position{x: -14.54658711, y: 1.336835893, z: 13.30508998}, starDetails:classK},
	{ starposition: position{x: -6.16961117, y: -9.937963822, z: 10.69984781}, starDetails:classF},
	{ starposition: position{x: 8.402497424, y: 3.997676339, z: 8.560542824}, starDetails:classG},
	{ starposition: position{x: -10.81817169, y: 7.336354957, z: 12.05899275}, starDetails:classK},
	{ starposition: position{x: -8.073792983, y: 0.6442285059, z: 7.487633254}, starDetails:classM},
	{ starposition: position{x: -0.4830936881, y: -6.879477323, z: 6.517077739}, starDetails:classM},
	{ starposition: position{x: -12.03251552, y: 2.594210175, z: 11.64753448}, starDetails:classM},
	{ starposition: position{x: -3.405342435, y: 0.8248546325, z: 3.327846191}, starDetails:classM},
	{ starposition: position{x: -2.422722063, y: 11.71458755, z: 11.38258093}, starDetails:classG},
	{ starposition: position{x: -1.146258143, y: -5.119798091, z: 5.010559952}, starDetails:classM},
	{ starposition: position{x: -6.02507909, y: -11.62398455, z: 12.5635011}, starDetails:classK},
	{ starposition: position{x: 2.557433221, y: 0.2051188713, z: 2.479516086}, starDetails:classM},
	{ starposition: position{x: 3.428727968, y: -1.133305264, z: 3.528316716}, starDetails:classM},
	{ starposition: position{x: 5.879734154, y: -6.700870235, z: 8.760023356}, starDetails:classM},
	{ starposition: position{x: -5.782312563, y: -5.829075397, z: 8.333365975}, starDetails:classM},
	{ starposition: position{x: -10.79856491, y: -7.380496165, z: 13.28432944}, starDetails:classM},
	{ starposition: position{x: -10.16891215, y: 4.138913453, z: 11.18299208}, starDetails:classK},
	{ starposition: position{x: 0.8623718427, y: -11.9442627, z: 12.21049762}, starDetails:classM},
	{ starposition: position{x: -0.9161551301, y: -4.321944795, z: 4.522507587}, starDetails:classK},
	{ starposition: position{x: 7.290402718, y: 7.922645574, z: 11.04586039}, starDetails:classM},
	{ starposition: position{x: 1.657122432, y: -10.67348937, z: 11.08525892}, starDetails:classM},
	{ starposition: position{x: 8.015074633, y: 0.1808148894, z: 8.240418588}, starDetails:classM},
	{ starposition: position{x: 8.187965322, y: 0.2026580979, z: 8.426112109}, starDetails:classK},
	{ starposition: position{x: 9.640489513, y: -9.418141915, z: 13.90004126}, starDetails:classK},
	{ starposition: position{x: 1.688458502, y: 8.827823613, z: 9.306870631}, starDetails:classM},
	{ starposition: position{x: -10.75067633, y: 6.942526812, z: 13.2620196}, starDetails:classG},
	{ starposition: position{x: -3.995245333, y: 10.51065691, z: 11.67973015}, starDetails:classM},
	{ starposition: position{x: -8.328525835, y: -3.857243912, z: 9.566294177}, starDetails:classM},
	{ starposition: position{x: -0.08743264181, y: -9.318224197, z: 9.849785143}, starDetails:classM},
	{ starposition: position{x: 11.89581639, y: -0.06666347666, z: 12.63661684}, starDetails:classM},
	{ starposition: position{x: 9.5447439, y: 5.077664423, z: 11.59435888}, starDetails:classD},
	{ starposition: position{x: -3.047907623, y: -11.91504465, z: 13.22562595}, starDetails:classG},
	{ starposition: position{x: -3.719006803, y: 10.82605965, z: 12.37917648}, starDetails:classG},
	{ starposition: position{x: -3.183215744, y: -11.42135136, z: 12.87663503}, starDetails:classK},
	{ starposition: position{x: -3.254738945, y: -11.71406564, z: 13.20580876}, starDetails:classK},
	{ starposition: position{x: -2.843318732, y: 12.42868147, z: 13.85046501}, starDetails:classK},
	{ starposition: position{x: -5.97474005, y: -6.175943227, z: 9.428379809}, starDetails:classG},
	{ starposition: position{x: -6.552411396, y: -2.377102568, z: 7.68123079}, starDetails:classK},
	{ starposition: position{x: 6.500733357, y: 4.647246654, z: 8.837614018}, starDetails:classM},
	{ starposition: position{x: -6.89599058, y: 6.943455401, z: 10.88464876}, starDetails:classA},
	{ starposition: position{x: -8.666527931, y: 4.884745336, z: 11.08464091}, starDetails:classM},
	{ starposition: position{x: -2.170821124, y: -4.883693096, z: 6.009850911}, starDetails:classM},
	{ starposition: position{x: 10.81504576, y: -0.5647570367, z: 12.46515651}, starDetails:classM},
	{ starposition: position{x: 5.531759544, y: 4.816946163, z: 8.506367316}, starDetails:classF},
	{ starposition: position{x: -2.818441603, y: 1.44553935, z: 3.702859969}, starDetails:classK},
	{ starposition: position{x: 4.632072959, y: 5.013056637, z: 8.023712178}, starDetails:classG},
	{ starposition: position{x: -4.924871574, y: 11.89983211, z: 15.17133603}, starDetails:classF},
	{ starposition: position{x: 2.811385832, y: 10.2839629, z: 12.63812962}, starDetails:classM},
	{ starposition: position{x: -7.95813707, y: -4.231135776, z: 10.72275133}, starDetails:classK},
	{ starposition: position{x: 4.860026228, y: -10.85933767, z: 14.28992363}, starDetails:classF},
	{ starposition: position{x: 3.229821644, y: 11.79789718, z: 15.07547207}, starDetails:classM},
	{ starposition: position{x: -5.861559022, y: -9.16637441, z: 13.4593045}, starDetails:classM},
	{ starposition: position{x: 8.738940764, y: -6.417522973, z: 13.64829901}, starDetails:classM},
	{ starposition: position{x: -6.697612547, y: 5.007169434, z: 10.58051046}, starDetails:classF},
	{ starposition: position{x: 1.500371651, y: -10.06125722, z: 12.88985072}, starDetails:classK},
	{ starposition: position{x: -9.148492872, y: -1.788578201, z: 11.82864845}, starDetails:classK},
	{ starposition: position{x: -7.253685847, y: -5.328377696, z: 11.45876047}, starDetails:classF},
	{ starposition: position{x: -2.065743916, y: -11.10252654, z: 14.68957085}, starDetails:classK},
	{ starposition: position{x: -2.851440552, y: 2.513075577, z: 4.987122151}, starDetails:classK},
	{ starposition: position{x: -2.813859121, y: 2.480652572, z: 4.922003022}, starDetails:classM},
	{ starposition: position{x: 2.161321858, y: 5.755995413, z: 8.12809685}, starDetails:classK},
	{ starposition: position{x: -6.346711567, y: -5.025123066, z: 10.70698159}, starDetails:classK},
	{ starposition: position{x: -4.791811904, y: -9.112469088, z: 13.63478635}, starDetails:classM},
	{ starposition: position{x: 2.056445707, y: 8.153501986, z: 11.21264237}, starDetails:classM},
	{ starposition: position{x: 0.5924812916, y: 7.260363665, z: 9.838087696}, starDetails:classK},
	{ starposition: position{x: 0.5982980232, y: 7.400642631, z: 10.03075026}, starDetails:classM},
	{ starposition: position{x: -1.540535786, y: -3.517909145, z: 5.345419186}, starDetails:classM},
	{ starposition: position{x: 4.773729714, y: -7.86253536, z: 12.86526512}, starDetails:classM},
	{ starposition: position{x: -8.884455897, y: -0.4789787395, z: 12.46697609}, starDetails:classM},
	{ starposition: position{x: -3.049578089, y: 6.900731743, z: 10.71477418}, starDetails:classM},
	{ starposition: position{x: 4.149480753, y: 1.272888297, z: 6.181214401}, starDetails:classG},
	{ starposition: position{x: -6.649601506, y: 2.732984351, z: 10.65083985}, starDetails:classF},
	{ starposition: position{x: -8.799672891, y: 3.567738846, z: 14.68636304}, starDetails:classM},
	{ starposition: position{x: 3.464759711, y: -0.7167963565, z: 5.483297469}, starDetails:classK},
	{ starposition: position{x: 9.613651755, y: 3.531630378, z: 15.96971842}, starDetails:classK},
	{ starposition: position{x: 1.971468174, y: -0.8367679319, z: 3.387440047}, starDetails:classM},
	{ starposition: position{x: 3.098175306, y: 0.6738832719, z: 5.038079837}, starDetails:classG},
	{ starposition: position{x: 6.89619613, y: -1.076533855, z: 11.10723153}, starDetails:classM},
	{ starposition: position{x: 4.652544163, y: 5.362637549, z: 11.43626083}, starDetails:classM},
	{ starposition: position{x: 9.446093014, y: 2.157731154, z: 15.68911428}, starDetails:classM},
	{ starposition: position{x: 5.443707559, y: 3.123507985, z: 10.25044653}, starDetails:classM},
	{ starposition: position{x: 3.373224946, y: -8.030233329, z: 14.26190424}, starDetails:classK},
	{ starposition: position{x: 0.0114626627, y: 7.082140063, z: 11.59952233}, starDetails:classM},
	{ starposition: position{x: 1.073950276, y: 2.63056632, z: 4.725440708}, starDetails:class},
	{ starposition: position{x: -1.733625809, y: -5.369389023, z: 9.411603457}, starDetails:classD},
	{ starposition: position{x: 8.555330335, y: 0.3424271284, z: 14.33485858}, starDetails:classF},
	{ starposition: position{x: 8.574672576, y: -1.077563557, z: 14.47730764}, starDetails:classK},
	{ starposition: position{x: 0.3301092924, y: -1.746699155, z: 3.032589608}, starDetails:classK},
	{ starposition: position{x: 0.3349818317, y: -1.772687634, z: 3.078148108}, starDetails:classK},
	{ starposition: position{x: -5.613061602, y: -7.678461019, z: 16.53227159}, starDetails:classK},
	{ starposition: position{x: -1.755388545, y: 7.92697494, z: 14.25410256}, starDetails:classM},
	{ starposition: position{x: -6.020181539, y: -6.251597461, z: 15.26797144}, starDetails:classM},
	{ starposition: position{x: -1.209849541, y: 5.033511006, z: 9.291139281}, starDetails:classM},
	{ starposition: position{x: 8.728545436, y: 2.058166778, z: 16.26115425}, starDetails:classF},
	{ starposition: position{x: -7.017288147, y: 6.189700321, z: 17.17876294}, starDetails:classF},
	{ starposition: position{x: -4.202043102, y: -2.366158489, z: 8.878986291}, starDetails:classK},
	{ starposition: position{x: -0.7113436629, y: -6.647776145, z: 12.40782397}, starDetails:classM},
	{ starposition: position{x: -5.507951563, y: 7.182779494, z: 16.8298259}, starDetails:classM},
	{ starposition: position{x: 4.469332192, y: -5.083404669, z: 12.64314831}, starDetails:classK},
	{ starposition: position{x: -0.7233534302, y: -6.601569379, z: 12.42502262}, starDetails:classG},
	{ starposition: position{x: 2.257921115, y: -2.393330264, z: 6.229128389}, starDetails:classM},
	{ starposition: position{x: 0.3949170362, y: 6.364940903, z: 12.11522009}, starDetails:classM},
	{ starposition: position{x: 4.494411379, y: 1.259778114, z: 8.907452172}, starDetails:classK},
	{ starposition: position{x: 5.249265594, y: -4.460514596, z: 13.28094754}, starDetails:classA},
	{ starposition: position{x: -4.11005342, y: 2.465885642, z: 9.322176387}, starDetails:classM},
	{ starposition: position{x: -2.094477631, y: 4.790206419, z: 10.23496792}, starDetails:classM},
	{ starposition: position{x: 5.328903619, y: 2.590089594, z: 12.05439222}, starDetails:classK},
	{ starposition: position{x: 3.919344339, y: 1.991287823, z: 8.955179241}, starDetails:classK},
	{ starposition: position{x: 6.328736485, y: 1.907817825, z: 13.51698337}, starDetails:classK},
	{ starposition: position{x: -7.85036219, y: -0.7910980924, z: 16.19860953}, starDetails:classM},
	{ starposition: position{x: -3.857662733, y: 4.63028708, z: 12.93649723}, starDetails:classG},
	{ starposition: position{x: -1.748067583, y: -6.098511713, z: 13.68887499}, starDetails:classF},
	{ starposition: position{x: 2.019551899, y: -2.61951525, z: 7.240812278}, starDetails:classM},
	{ starposition: position{x: -3.665484671, y: 0.6438424394, z: 8.298764961}, starDetails:classM},
	{ starposition: position{x: -3.041098116, y: -2.605774782, z: 9.018568704}, starDetails:classM},
	{ starposition: position{x: -4.03233132, y: -0.8765438306, z: 9.316565794}, starDetails:classM},
	{ starposition: position{x: -4.551711617, y: 4.880962636, z: 15.08287688}, starDetails:classK},
	{ starposition: position{x: 3.899462863, y: -5.699793262, z: 16.15423608}, starDetails:classG},
	{ starposition: position{x: -5.007847004, y: -0.516344437, z: 11.96969353}, starDetails:class},
	{ starposition: position{x: -2.000457611, y: 6.558774446, z: 16.31344421}, starDetails:classM},
	{ starposition: position{x: 3.87949654, y: 0.5529435133, z: 9.338747293}, starDetails:classM},
	{ starposition: position{x: -1.796591225, y: -3.713323597, z: 9.831797259}, starDetails:classM},
	{ starposition: position{x: -1.807128343, y: -3.736867745, z: 9.901671064}, starDetails:classM},
	{ starposition: position{x: -3.383818082, y: 4.161893521, z: 12.81967884}, starDetails:classM},
	{ starposition: position{x: -0.7507542234, y: -4.879856199, z: 11.80679784}, starDetails:classK},
	{ starposition: position{x: -1.568203245, y: 5.431098503, z: 14.19711851}, starDetails:classM},
	{ starposition: position{x: -0.1714854634, y: -1.662275427, z: 4.208308009}, starDetails:classM},
	{ starposition: position{x: 3.442859427, y: 5.302243017, z: 16.19182028}, starDetails:classK},
	{ starposition: position{x: 3.441955776, y: 5.30127907, z: 16.19232802}, starDetails:classM},
	{ starposition: position{x: 2.989754706, y: 5.717777783, z: 17.29609326}, starDetails:classG},
	{ starposition: position{x: 0.7857595076, y: -1.843455313, z: 5.40729638}, starDetails:classK},
	{ starposition: position{x: -3.311088401, y: 2.26291195, z: 11.03944828}, starDetails:classM},
	{ starposition: position{x: -3.253750771, y: 2.217868699, z: 10.84244434}, starDetails:classM},
	{ starposition: position{x: 5.495087313, y: 0.6475249005, z: 15.32076871}, starDetails:classM},
	{ starposition: position{x: 2.465605816, y: 0.6893746432, z: 7.732506981}, starDetails:classM},
	{ starposition: position{x: -0.6128074674, y: -5.884746428, z: 18.07872925}, starDetails:classK},
	{ starposition: position{x: 0.2192343176, y: -2.381464166, z: 7.694267431}, starDetails:classF},
	{ starposition: position{x: -4.095416309, y: 0.8106882999, z: 14.07156007}, starDetails:classK},
	{ starposition: position{x: 2.691964788, y: -0.08919956517, z: 10.4482278}, starDetails:classK},
	{ starposition: position{x: -1.798796327, y: -2.598881737, z: 12.6717131}, starDetails:classM},
	{ starposition: position{x: -3.206788473, y: 2.106374403, z: 15.44138596}, starDetails:classM},
	{ starposition: position{x: 1.892448417, y: 3.321638622, z: 15.51814673}, starDetails:classK},
	{ starposition: position{x: 3.621273254, y: 1.353699042, z: 16.36765029}, starDetails:classK},
	{ starposition: position{x: 4.349561935, y: 0.4478951134, z: 19.2290969}, starDetails:classK},
	{ starposition: position{x: 2.005714588, y: -3.122090666, z: 16.38325249}, starDetails:classK},
	{ starposition: position{x: 2.942365452, y: -0.2658384996, z: 13.47299218}, starDetails:classK},
	{ starposition: position{x: -1.055816622, y: 0.05693719786, z: 5.286717812}, starDetails:classM},
	{ starposition: position{x: 3.707232317, y: -0.6631359987, z: 19.38081578}, starDetails:classG},
	{ starposition: position{x: -0.6478373901, y: 3.167160253, z: 17.55560611}, starDetails:classF},
	{ starposition: position{x: 1.527554449, y: 1.842238834, z: 13.52618479}, starDetails:classM},
	{ starposition: position{x: -1.400670546, y: 2.518718399, z: 16.79906802}, starDetails:classG},
	{ starposition: position{x: -2.292247579, y: -1.435189988, z: 16.33390998}, starDetails:classM},
	{ starposition: position{x: 3.167332013, y: 0.1885926384, z: 19.30277338}, starDetails:classM},
	{ starposition: position{x: -2.487605063, y: -1.898070545, z: 19.34100008}, starDetails:classG},
	{ starposition: position{x: -0.05809926944, y: 1.288473857, z: 9.306900725}, starDetails:classM},
	{ starposition: position{x: 2.080750445, y: -0.6806496723, z: 19.81963548}, starDetails:classK},
}

func getStarDetails() []*star {
	stars := make([]*star, len(starPositionsDetails))
	for i, positionDetails := range starPositionsDetails {
		stars[i] = &star{}
		stars[i].x = positionDetails.starposition.x
		stars[i].y = positionDetails.starposition.y
		stars[i].z = positionDetails.starposition.z
		stars[i].brightColor = positionDetails.starDetails.brightColor
		stars[i].dimColor = positionDetails.starDetails.dimColor
		stars[i].pixels = 5
		stars[i].radii = 500000.0
		stars[i].class = positionDetails.starDetails.class
	}

	return stars
}

func getSectorDetails(fromSector sector) (result []*star) {
	result = getStarDetails()

	return
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

var (
	stars []*star
	lines []*simpleLine

	sName       = "sphere"
	sphereModel *gi3d.Sphere

	rendered      = false
	connectedStar int
	highWater     int
)

func renderStars(sc *gi3d.Scene) {
	if !rendered {
		stars = make([]*star, 0)
		id := 0
		for x := uint32(0); x < 2; x++ {
			for y := uint32(0); y < 2; y++ {
				for z := uint32(0); z < 2; z++ {
					sector := sector{x: x, y: y, z: z}
					for _, star := range getSectorDetails(sector) {
						star.id = id
						id++
						stars = append(stars, star)
					}
				}
			}
		}
		if len(stars) > 0 {
			sphereModel = &gi3d.Sphere{}
			sphereModel.Reset()
			sphereModel = gi3d.AddNewSphere(sc, sName, 0.002, 24)
			lines = make([]*simpleLine, 0)
			sName = "sphere"
			for _, star := range stars {
				starSphere := gi3d.AddNewSolid(sc, sc, sName, sphereModel.Name())
				starSphere.Pose.Pos.Set(star.x+offsets.x, star.y+offsets.y, star.z+offsets.z)
				starSphere.Mat.Color.SetUInt8(star.brightColor.R, star.brightColor.G, star.brightColor.B, star.brightColor.A)
			}
			for id, star := range stars {
				for _, jump := range checkForJumps(stars, star, id) {
					lines = append(lines, jump)
					if jump.jumpInfo.distance < 3.0 {
						jumpsByStar[star.id] = append(jumpsByStar[star.id], jump.jumpInfo)
						if star.id == jump.jumpInfo.s2ID {
							jumpsByStar[jump.jumpInfo.s1ID] = append(jumpsByStar[jump.jumpInfo.s1ID], jump.jumpInfo)
						} else {
							jumpsByStar[jump.jumpInfo.s2ID] = append(jumpsByStar[jump.jumpInfo.s2ID], jump.jumpInfo)
						}
					}
				}
			}

			if !fastest {
				rendered = true
				highWater = -1
				for lNumber := 0; lNumber < len(stars); lNumber++ {
					tJumps := traceJumps(lNumber)
					if len(tJumps) > highWater {
						highWater = len(tJumps)
						connectedStar = lNumber
					}
				}
				f, err := os.Create("traveler-report.csv")

				if err == nil {
					alreadyPrinted := make([]int, 0)
					_, err := f.Write([]byte(csvTextHdr))
					if err != nil {
						os.Exit(-1)
					}
					for _, nextJump := range traceJumps(connectedStar) {
						if !contains(alreadyPrinted, nextJump.s1ID) && nextJump.s1ID > -1 {
							_, err := f.Write([]byte(worldFromStar(nextJump.s1ID).worldCSV))
							if err != nil {
								os.Exit(-1)
							}
							alreadyPrinted = append(alreadyPrinted, nextJump.s1ID)
						}
						if !contains(alreadyPrinted, nextJump.s2ID) && nextJump.s2ID > -1  {
							_, err := f.Write([]byte(worldFromStar(nextJump.s2ID).worldCSV))
							if err != nil {
								os.Exit(-1)
							}
							alreadyPrinted = append(alreadyPrinted, nextJump.s2ID)
						}

					}
				}

				if !faster {
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
				}
			}
			// fastest case
			for id, lin := range lines {
				thickness := float32(0.00010)
				if lin.jumpInfo.color.A < math.MaxUint8-47 {
					thickness = 0.00012
				} else if lin.jumpInfo.color.A < math.MaxUint8-39 {
					thickness = 0.00015
				}
				if lin.jumpInfo.s1ID != lin.jumpInfo.s2ID {
					lin.lines = gi3d.AddNewLines(sc, "Lines-"+strconv.Itoa(lin.jumpInfo.s1ID)+"-"+strconv.Itoa(lin.jumpInfo.s2ID),
						[]mat32.Vec3{
							{X: lin.from.x + offsets.x, Y: lin.from.y + offsets.y, Z: lin.from.z + offsets.z},
							{X: lin.to.x + offsets.x, Y: lin.to.y + offsets.y, Z: lin.to.z + offsets.z},
						},
						mat32.Vec2{X: thickness, Y: thickness},
						gi3d.OpenLines,
					)
					solidLine := gi3d.AddNewSolid(sc, sc, "Lines-"+strconv.Itoa(id), lin.lines.Name())
					// solidLine.Pose.Pos.Set(lin.from.x - .5, lin.from.y - .5, lin.from.z + 8)
					// lns.Mat.Color.SetUInt8(255, 255, 0, 128)
					solidLine.Mat.Color = lin.jumpInfo.color
				}
			}
		}
	}
}

func checkForJumps(stars []*star, star *star, id int) (result []*simpleLine) {
	result = make([]*simpleLine, 0)
	for innerId, innerStar := range stars {
		if innerId == id {
			continue
		}
		jumpColor := checkFor1jump(star, innerStar)
		if jumpColor.color.A > 0 {
			// symmetric, so no copies
			result = addIfNew(result, jumpColor)
		}
	}
	closest := []*simpleLine{&noLine, &noLine, &noLine}
	if len(result) > 3 {
		for _, nextSimpleLine := range result {
			if nextSimpleLine.jumpInfo.distance < closest[0].jumpInfo.distance {
				closest[2] = closest[1]
				closest[1] = closest[0]
				closest[0] = nextSimpleLine
			} else if nextSimpleLine.jumpInfo.distance < closest[1].jumpInfo.distance {
				closest[2] = closest[1]
				closest[1] = nextSimpleLine
			} else if nextSimpleLine.jumpInfo.distance < closest[2].jumpInfo.distance {
				closest[2] = nextSimpleLine
			}
		}
		result = closest
	}

	return
}

func checkFor1jump(s1 *star, s2 *star) (result *jump) {
	jumpLength := distance(s1, s2) * 100 * parsecsPerLightYear
	delta := int(jumpLength)
	if delta < len(jumpColors) {
		if s1.id != s2.id {
			result = &jump{jumpColors[delta], jumpColors[delta], delta, jumpLength, s1.id, s2.id}
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

func addIfNew(soFar []*simpleLine, jump *jump) (result []*simpleLine) {
	result = soFar
	if jump.s1ID >= jump.s2ID {
		return
	}
	already := false
	for _, line := range result {
		if (line.jumpInfo.s1ID == jump.s1ID &&
			line.jumpInfo.s2ID == jump.s2ID) ||
			(line.jumpInfo.s1ID == jump.s2ID &&
				line.jumpInfo.s2ID == jump.s1ID) {
			already = true
			break
		}
	}
	if !already {
		nextLine := &simpleLine{
			from:     position{x: stars[jump.s1ID].x, y: stars[jump.s1ID].y, z: stars[jump.s1ID].z},
			to:       position{x: stars[jump.s2ID].x, y: stars[jump.s2ID].y, z: stars[jump.s2ID].z},
			jumpInfo: jump,
		}
		result = append(result, nextLine)
	}
	return
}

func contains(soFar []int, next int) (yes bool) {
	yes = false
	for _, sID := range soFar {
		if sID == next {
			yes = true
			break
		}
	}
	return
}
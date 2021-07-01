package main

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strconv"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"

	"github.com/spaolacci/murmur3"
)

type world struct {
	starID                int
	starPort              string
	scout                 bool
	navy                  bool
	military              bool
	gasGiants             int
	size                  int
	sizeBase              int
	atmosphereDescription atmosphereDetails
	atmosphereBase        int
	hydro                 int
	hydroBase             int
	population            uint64
	popBase               int
	lawLevel              string
	lawBase               int
	government            string
	governmentBase        int
	techLevel             string
	techLevelBase         int
	worldHeader           string
	worldCSV              string
	worldLayout           *gi.Layout
	SystemDetails         *gi.Label
	jumpButtons           []*gi.Button
	jumps                 []int
}

var workingWorld = &world{}

type atmosphereDetails struct {
	description string
	base        int
	tainted     bool
	trace       bool
	veryThin    bool
	thin        bool
	standard    bool
	dense       bool
	exotic      bool
	corrosive   bool
	insidious   bool
}

var (
	noAtmosphere    = atmosphereDetails{description: "No atmosphere", base: 0, tainted: false, trace: false, veryThin: false, thin: false, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	traceAtmosphere = atmosphereDetails{description: "Trace", base: 1, tainted: false, trace: true, veryThin: false, thin: false, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	veryThinTainted = atmosphereDetails{description: "Very thin - tainted", base: 2, tainted: true, trace: false, veryThin: true, thin: false, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	veryThin        = atmosphereDetails{description: "Very thin", base: 3, tainted: false, trace: false, veryThin: true, thin: false, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	thinTainted     = atmosphereDetails{description: "Thin - tainted", base: 4, tainted: true, trace: false, veryThin: false, thin: true, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	thin            = atmosphereDetails{description: "Thin", base: 5, tainted: false, trace: false, veryThin: false, thin: true, standard: false, dense: false, exotic: false, corrosive: false, insidious: false}
	standard        = atmosphereDetails{description: "Standard", base: 6, tainted: false, trace: false, veryThin: false, thin: false, standard: true, dense: false, exotic: false, corrosive: false, insidious: false}
	standardTainted = atmosphereDetails{description: "Standard - tainted", base: 7, tainted: true, trace: false, veryThin: false, thin: false, standard: true, dense: false, exotic: false, corrosive: false, insidious: false}
	dense           = atmosphereDetails{description: "Dense", base: 8, tainted: false, trace: false, veryThin: false, thin: false, standard: false, dense: true, exotic: false, corrosive: false, insidious: false}
	denseTainted    = atmosphereDetails{description: "Dense - tainted", base: 9, tainted: true, trace: false, veryThin: false, thin: false, standard: false, dense: true, exotic: false, corrosive: false, insidious: false}
	exotic          = atmosphereDetails{description: "Exotic", base: 10, tainted: false, trace: false, veryThin: false, thin: false, standard: false, dense: false, exotic: true, corrosive: false, insidious: false}
	corrosive       = atmosphereDetails{description: "Corrosive", base: 11, tainted: false, trace: false, veryThin: false, thin: false, standard: false, dense: false, exotic: false, corrosive: true, insidious: false}
	insidious       = atmosphereDetails{description: "Insidious", base: 12, tainted: false, trace: false, veryThin: false, thin: false, standard: false, dense: false, exotic: false, corrosive: false, insidious: true}

	atmospheres = []atmosphereDetails{
		noAtmosphere, traceAtmosphere, veryThinTainted, veryThin, thinTainted, thin, standard, standardTainted, dense, denseTainted, exotic, corrosive, insidious, insidious, insidious, insidious, insidious,
	}
)

func worldHash(fromStar *star) *rand.Rand {
	id := murmur3.New64()
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(65535*fromStar.x-float32(int(fromStar.x))))
	_, err := id.Write(buf)
	if err != nil {
		print("Failed to hash part 1")
	}

	binary.LittleEndian.PutUint32(buf, uint32(65535*fromStar.y-float32(int(fromStar.y))))
	_, err = id.Write(buf)
	if err != nil {
		print("Failed to hash part two")
	}

	binary.LittleEndian.PutUint32(buf, uint32(65535*fromStar.z-float32(int(fromStar.z))))
	_, err = id.Write(buf)
	if err != nil {
		print("Failed to hash part 3")
	}

	return rand.New(rand.NewSource(int64(id.Sum64())))
}

func worldFr omStar(fromStarID int) (newWorld *world) {
	random1s := worldHash(stars[fromStarID])

	starPort := getStarPort(random1s)
	size, sizeBase := getSize(random1s)
	atmosphereDescription, atmosphereBase := getAtmosphere(random1s, sizeBase)
	hydro, hydroBase := getHydro(random1s, atmosphereBase)
	population, popBase := getPopulation(random1s)
	lawLevel, lawBase := getLawLevel(random1s, popBase)
	government, governmentBase := getGovernment(random1s, popBase)
	techLevel, tl := getTechLevel(random1s, starPort, size, atmosphereBase, hydroBase, popBase, governmentBase)

	header := fmt.Sprintf(hdrText, fromStarID, starPort, size, atmosphereDescription.description, size, hydro,
		population, government, lawBase, tl)
	jumps := ""
	for _, jump  := range jumpsByStar[fromStarID] {
		if jump.s1ID == fromStarID {
			if jump.s2ID > -1 {
				jumps += fmt.Sprintf("jump to %d is %f parsecs, ", jump.s2ID, jump.distance)
			}
		} else {
			if jump.s1ID > -1 {
				jumps += fmt.Sprintf("jump to %d is %f parsecs, ", jump.s1ID, jump.distance)
			}
		}
	}
	worldCSV := fmt.Sprintf(csvText, fromStarID, stars[fromStarID].x, stars[fromStarID].y, stars[fromStarID].z,
		starPort, size, atmosphereDescription.description, hydro,
		population, government, lawLevel, tl, jumps)
	newWorld = &world{
		starID:                fromStarID,
		starPort:              starPort,
		scout:                 getScout(random1s, starPort),
		navy:                  getNavy(random1s, starPort),
		military:              getMilitary(random1s, starPort, popBase, atmospheres[atmosphereBase].tainted),
		gasGiants:             getGasGiants(random1s),
		size:                  size,
		sizeBase:              sizeBase,
		atmosphereDescription: atmosphereDescription,
		atmosphereBase:        atmosphereBase,
		hydro:                 hydro,
		hydroBase:             hydroBase,
		population:            population,
		popBase:               popBase,
		lawLevel:              lawLevel,
		lawBase:               lawBase,
		government:            government,
		governmentBase:        governmentBase,
		techLevel:             techLevel,
		techLevelBase:         tl,
		worldHeader:           header,
		worldCSV:              worldCSV,
		SystemDetails:         workingWorld.SystemDetails,
	}

	return
}

func getStarPort(rand *rand.Rand) (portType string) {
	huh := twoD6(rand)
	switch huh {
	case 2, 3, 4:
		portType = "A"
	case 5, 6:
		portType = "B"
	case 7, 8:
		portType = "C"
	case 9:
		portType = "D"
	case 10, 11:
		portType = "E"
	case 12:
		portType = "X"
	default:
		portType = "S"
	}

	return
}

func getSize(rand *rand.Rand) (kilometers int, base int) {
	base = twoD6(rand) - 2
	kilometers = 1600 * base

	return
}

func getAtmosphereFromID(fromAtmosphereBase int) (result atmosphereDetails) {
	switch fromAtmosphereBase {
	default:
		return noAtmosphere
	case 0:
		return noAtmosphere
	case 1:
		return traceAtmosphere
	case 2:
		return veryThinTainted
	case 3:
		return veryThin
	case 4:
		return thinTainted
	case 5:
		return thin
	case 6:
		return standard
	case 7:
		return standardTainted
	case 8:
		return dense
	case 9:
		return denseTainted
	case 10:
		return exotic
	case 11:
		return corrosive
	case 12, 13, 14, 15, 16:
		return insidious
	}
}

func getAtmosphere(rand *rand.Rand, size int) (result atmosphereDetails, base int) {
	base = twoD6(rand) + size - 7
	if base < 0 {
		base = 0
	}
	result = getAtmosphereFromID(base)
	return
}

func getHydro(rand *rand.Rand, atmosphere int) (percent int, base int) {
	percent = 10 * (twoD6(rand) + atmosphere - 7)
	if percent < 0 {
		percent = 0.0
	} else if percent > 100 {
		percent = 100
	}
	base = percent / 10

	return
}

func getPopulation(rand *rand.Rand) (total uint64, base int) {
	base = twoD6(rand) - 2.0
	log := rand.Float32()
	if base < 0 {
		base = 0
	} else if base > 10 {
		base = 10
	}
	if base < 1 {
		total = 0

		return
	}
	total = uint64(math.Exp(float64(float32(base)+log) * math.Log(10)))

	return
}

var govByBase = []string{
	"No government",
	"Company/Corporation",
	"Participating Democracy",
	"Self-Perpetuating Oligarchy",
	"Representative Democracy",
	"Feudal Technocracy",
	"Captive Government",
	"Balkanization",
	"Civil Service Bureaucracy",
	"Impersonal Bureaucracy",
	"Charismatic Dictator",
	"Non-Charismatic Leader",
	"Charismatic Oligarchy",
	"Religious Dictatorship",
}

func getGovernment(rand *rand.Rand, popBase int) (description string, base int) {
	base = d6(rand) + d6(rand) + popBase - 7
	if base < 0 {
		base = 0
	} else if base >= len(govByBase) {
		base = len(govByBase) - 1
	}
	description = govByBase[base]

	return
}

var lawLevelByBase = []string{
	"No Prohibitions",
	"Body pistols explosives & poison gas prohibited",
	"Portable energy weapons prohibited",
	"Military weapons (automatics) prohibited",
	"Light assault weapons prohibited",
	"Personal firearms prohibited",
	"Most firearms (except shotgun) prohibited all weapons discouraged",
	"Shotguns prohibited",
	"Long bladed weapons prohibited",
	"Possession of any weapon outside residence prohibited",
}

func getLawLevel(rand *rand.Rand, govBase int) (description string, base int) {
	base = d6(rand) + d6(rand) + govBase - 7
	if base < 0 {
		base = 0
	} else if base >= len(lawLevelByBase) {
		base = len(lawLevelByBase) - 1
	}
	description = lawLevelByBase[base]

	return
}

func getTechLevel(rand *rand.Rand, starPort string, size int, atm int, hydro int, pop int, gov int) (techLevel string, tl int) {
	diceModifier := 0
	switch starPort {
	case "A":
		diceModifier = 6
	case "B":
		diceModifier = 4
	case "C":
		diceModifier = 2
	case "X":
		diceModifier = -4
	}
	switch size {
	case 0, 1:
		diceModifier += 2
	case 2, 3, 4:
		diceModifier++
	}
	if atm < 4 || atm > 9 {
		diceModifier++
	}
	if hydro > 8 {
		diceModifier += hydro - 8
	}
	if pop > 0 && pop < 6 {
		diceModifier++
	}
	if pop > 8 {
		diceModifier += 2 * (pop - 8)
	}
	if gov == 0 || gov == 5 {
		diceModifier++
	} else if gov == 13 {
		diceModifier -= 2
	}
	tl = d6(rand) + diceModifier
	if tl < 1 {
		tl = 1
	}
	if tl > 9 {
		switch tl {
		case 10:
			techLevel = "A"
		case 11:
			techLevel = "B"
		case 12:
			techLevel = "C"
		case 13:
			techLevel = "D"
		case 14:
			techLevel = "E"
		case 15:
			techLevel = "F"
		case 16:
			techLevel = "G"
		case 17:
			techLevel = "H"
		case 18:
			techLevel = "J"
		case 19:
			techLevel = "K"
		case 20:
			techLevel = "L"
		case 21:
			techLevel = "M"
		case 22:
			techLevel = "N"
		case 23:
			techLevel = "O"
		case 24:
			techLevel = "P"
		case 25:
			techLevel = "Q"
		case 26:
			techLevel = "R"
		case 27:
			techLevel = "S"
		case 28:
			techLevel = "T"
		case 29:
			techLevel = "U"
		case 30:
			techLevel = "V"
		case 31:
			techLevel = "W"
		case 32:
			techLevel = "X"
		case 33:
			techLevel = "Y"
		case 34:
			techLevel = "Z"
		default:
			techLevel = "9"
		}
	} else {
		techLevel = strconv.Itoa(tl)
	}

	return
}

func twoD6(rand *rand.Rand) (result int) {
	result = d6(rand) + d6(rand)

	return
}

func d6(rand *rand.Rand) (result int) {
	result = rand.Intn(6) + 1

	return
}

func getScout(rand *rand.Rand, starPort string) (scout bool) {
	scout = false
	mod := 0
	switch starPort {
	case "A":
		mod = -3
	case "B":
		mod = -2
	case "C":
		mod = -1
	case "E", "X":
		mod = -99
	}
	if twoD6(rand)+mod > 6 {
		scout = true
	}

	return
}

func getNavy(rand *rand.Rand, starPort string) (navy bool) {
	navy = false
	if starPort != "C" && starPort != "D" && starPort != "E" && starPort != "X" {
		if twoD6(rand) > 6 {
			navy = true
		}
	}

	return
}

func getGasGiants(rand *rand.Rand) (gasGiants int) {
	if twoD6(rand) < 10 {
		switch twoD6(rand) {
		case 2, 3, 4, 5, 6, 7, 8, 9:
			gasGiants = 1
		case 10, 11:
			gasGiants = 2
		case 12, 13:
			gasGiants = 3
		default:
			gasGiants = 1
		}
	} else {
		gasGiants = 0
	}

	return
}

func getMilitary(rand *rand.Rand, starPort string, popBase int, tainted bool) (mil bool) {
	if popBase < 4 && (starPort == "A" || starPort == "B") ||
		popBase > 7 && (starPort == "A" || starPort == "B") {
		if tainted {
			if twoD6(rand) > 5 {
				mil = true
			} else {
				mil = false
			}
		} else {
			if twoD6(rand) > 8 {
				mil = true
			} else {
				mil = false
			}
		}
	} else {
		if twoD6(rand) > 9 {
			mil = true
		} else {
			mil = false
		}
	}

	return
}

func putWorldHeader(layout *gi.Layout) {
	workingWorld.worldLayout = layout
	workingWorld.SystemDetails = gi.AddNewLabel(layout, "SystemDetails", hdrText)
	workingWorld.SystemDetails.CurBgColor = gist.Color{R: 0, G: 0, B: 0, A: 255}
	workingWorld.SystemDetails.SetProp("white-space", gist.WhiteSpaceNormal)
	workingWorld.SystemDetails.SetProp("background-color", color.Opaque)
	workingWorld.SystemDetails.SetProp("text-align", gist.AlignLeft)
	workingWorld.SystemDetails.SetProp("vertical-align", gist.AlignTop)
	workingWorld.SystemDetails.SetProp("font-family", "Times New Roman, serif")
	workingWorld.SystemDetails.SetProp("font-size", "small")
	// SystemDetails.SetProp("letter-spacing", 2)
	workingWorld.SystemDetails.SetProp("line-height", 1.5)
}

func getWorldHeader() string {
	return workingWorld.SystemDetails.Text
}

func setWorldHeader(header string) {
	workingWorld.SystemDetails.SetText(header)
}



func maxTech() (results []*star) {
	techMax := -99
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.techLevelBase > techMax {
			techMax = world.techLevelBase
			results = append(empty, star)
		} else if world.techLevelBase == techMax {
			results = append(results, star)
		}
	}

	return
}


func starsByTech(tech int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.techLevelBase== tech {
			results = append(results, star)
		}
	}

	return
}

func starsTechAtLeast(tech int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.techLevelBase >= tech {
			results = append(results, star)
		}
	}

	return
}

func starsTechAtMost(tech int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.techLevelBase <= tech {
			results = append(results, star)
		}
	}

	return
}

func maxPop() (results []*star) {
	popMax := -99
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase > popMax {
			popMax = world.popBase
			results = append(empty, star)
		} else if world.popBase == popMax {
			results = append(results, star)
		}
	}

	return
}

func minPop() (results []*star) {
	popMin := 199
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase < popMin {
			popMin = world.popBase
			results = append(empty, star)
		} else if world.popBase == popMin {
			results = append(results, star)
		}
	}

	return
}

func starsByPop(pop int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase == pop {
			results = append(results, star)
		}
	}

	return
}

func starsPopAtLeast(pop int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase >= pop {
			results = append(results, star)
		}
	}

	return
}

func starsPopAtMost(pop int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase <= pop {
			results = append(results, star)
		}
	}

	return
}

func allStars() (results []*star) {
	results = stars

	return
}

func maxSize() (results []*star) {
	sizeMax := -99
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase > sizeMax {
			sizeMax = world.popBase
			results = append(empty, star)
		} else if world.popBase == sizeMax {
			results = append(results, star)
		}
	}

	return
}

func minSize() (results []*star) {
	sizeMin := 99
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.popBase < sizeMin {
			sizeMin = world.popBase
			results = append(empty, star)
		} else if world.popBase == sizeMin {
			results = append(results, star)
		}
	}

	return
}

func starsBySize(size int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.sizeBase == size {
			results = append(results, star)
		}
	}

	return
}

func starsSizeAtMost(size int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.sizeBase <= size {
			results = append(results, star)
		}
	}

	return
}

func starsSizeAtLeast(size int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.sizeBase >= size {
			results = append(results, star)
		}
	}

	return
}

func starHydroMax() (results []*star) {
	hydroMax := -100
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.hydroBase > hydroMax {
			hydroMax = world.hydroBase
			results = append(empty, star)
		} else if world.hydroBase == hydroMax {
			results = append(results, star)
		}
	}

	return
}

func starHydroMin() (results []*star) {
	hydroMin := 300
	empty :=  make([]*star, 0)
	results = empty
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.hydroBase < hydroMin {
			hydroMin = world.hydroBase
			results = append(empty, star)
		} else if world.hydroBase == hydroMin {
			results = append(results, star)
		}
	}

	return
}

func starHydroAtLeast(hydroMin int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.hydroBase >= hydroMin {
			results = append(results, star)
		}
	}

	return
}

func starHydroAtMost(hydroMax int) (results []*star) {
	results = make([]*star, 0)
	for _, star := range stars {
		world := worldFromStar(star.id)
		if world.hydroBase <= hydroMax {
			results = append(results, star)
		}
	}

	return
}

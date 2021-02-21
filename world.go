package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"strconv"

	"github.com/spaolacci/murmur3"
)

type world struct {
	starport       string
	scout          bool
	naval          bool
	military       bool
	gasGiant       bool
	gasGiants      int
	size           int
	sizeBase       int
	atmosphere     atmpsphere
	atmosphereBase int
	hydro          int
	hydroBase      int
	population     uint64
	popBase        int
	lawLevel       string
	lawBase        int
	government     string
	governmentBase int
	techLevel      string
	techLevelBase  int
}

type atmpsphere struct {
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

func worldHash(fromStar star) *rand.Rand {
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

func worldFromStar(fromStar star) (toWorld world) {
	random1s := worldHash(fromStar)
	starPort := starPort(random1s)

	scout := false
	mod := 0
	if starPort == "A" {
		mod = -3
	} else if starPort == "B" {
		mod = -2
	} else if starPort == "C" {
		mod = -1
	} else if starPort == "E" || starPort == "X" {
		mod = -99
	}
	if twoD6(random1s)+mod > 6 {
		scout = true
	}

	navy := false
	mod = 0
	if starPort == "C" || starPort == "D" || starPort == "E" || starPort == "X" {
		mod = -99
	}
	if twoD6(random1s)+mod > 6 {
		navy = true
	}

	gasGiant := false
	gasGiants := 0
	if twoD6(random1s) < 10 {
		gasGiant = true

		switch twoD6(random1s) {
		case 2, 3, 4, 5, 6, 7, 8, 9:
			gasGiants = 1
		case 10, 11:
			gasGiants = 2
		case 12, 13:
			gasGiants = 3
		default:
			gasGiants = 1
		}
	}

	km, size := size(random1s)
	atm, atmBase := atmosphere(random1s, size)
	hydro, hydroBase := hydrographics(random1s, size)
	pop, popBase := population(random1s)
	lawDescription, lawBase := lawLevel(random1s, popBase)
	govDescription, govBase := government(random1s, popBase)

	mil := false
	if (popBase < 4 && (starPort == "A" || starPort == "B")) ||
		(popBase > 7 && (starPort == "A" || starPort == "B") && !atm.tainted) {
		if twoD6(random1s) > 5 {
			mil = true
		}
	} else if (popBase < 4 && (starPort == "A" || starPort == "B")) ||
		(popBase > 7 && (starPort == "A" || starPort == "B")) {
		if twoD6(random1s) > 8 {
			mil = true
		}
	} else {
		if twoD6(random1s) > 9 {
			mil = true
		}
	}

	techLevel, tl := techLevel(random1s, starPort, size, atmBase, hydroBase, popBase, govBase)

	toWorld = world{
		starport:       starPort,
		scout:          scout,
		naval:          navy,
		military:       mil,
		gasGiant:       gasGiant,
		gasGiants:      gasGiants,
		size:           km,
		sizeBase:       size,
		atmosphere:     atm,
		atmosphereBase: atmBase,
		hydro:          hydro,
		hydroBase:      hydroBase,
		population:     pop,
		popBase:        popBase,
		lawLevel:       lawDescription,
		lawBase:        lawBase,
		government:     govDescription,
		governmentBase: govBase,
		techLevel:      techLevel,
		techLevelBase:  tl,
	}

	return
}

func starPort(rand *rand.Rand) (portType string) {
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

func size(rand *rand.Rand) (kilometers int, base int) {
	base = twoD6(rand) - 2
	kilometers = 1600 * base

	return
}

func atmosphere(rand *rand.Rand, size int) (result atmpsphere, base int) {
	randomAtmpsphere := twoD6(rand) + size - 7
	if randomAtmpsphere < 0 {
		randomAtmpsphere = 0
	}
	switch randomAtmpsphere {
	case 0:
		return atmpsphere{
			description: "No atmosphere",
			base:        0,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	default:
		return atmpsphere{
			description: "No atmosphere",
			base:        0,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 1:
		return atmpsphere{
			description: "Trace",
			base:        1,
			tainted:     false,
			trace:       true,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 2:
		return atmpsphere{
			description: "Very thin, tainted",
			base:        2,
			tainted:     true,
			trace:       false,
			veryThin:    true,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 3:
		return atmpsphere{
			description: "Very thin",
			base:        3,
			tainted:     false,
			trace:       false,
			veryThin:    true,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 4:
		return atmpsphere{
			description: "Thin, tainted",
			base:        4,
			tainted:     true,
			trace:       false,
			veryThin:    false,
			thin:        true,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 5:

		return atmpsphere{
			description: "Thin",
			base:        5,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        true,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 6:
		return atmpsphere{
			description: "Standard",
			base:        6,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    true,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 7:
		return atmpsphere{
			description: "Standard, tainted",
			base:        7,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    true,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 8:
		return atmpsphere{
			description: "Dense",
			base:        8,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       true,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 9:
		return atmpsphere{
			description: "Dense, tainted",
			base:        9,
			tainted:     true,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       true,
			exotic:      false,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 10:
		return atmpsphere{
			description: "Exotic",
			base:        10,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      true,
			corrosive:   false,
			insidious:   false,
		}, randomAtmpsphere
	case 11:
		return atmpsphere{
			description: "Corrosive",
			base:        11,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   true,
			insidious:   false,
		}, randomAtmpsphere
	case 12, 13, 14, 15, 16:
		return atmpsphere{
			description: "Insidious",
			base:        12,
			tainted:     false,
			trace:       false,
			veryThin:    false,
			thin:        false,
			standard:    false,
			dense:       false,
			exotic:      false,
			corrosive:   false,
			insidious:   true,
		}, randomAtmpsphere
	}
}

func hydrographics(rand *rand.Rand, size int) (percent int, base int) {
	percent = 10 * (twoD6(rand) + size - 7)
	if percent < 0 {
		percent = 0.0
	} else if percent > 100 {
		percent = 100
	}
	base = percent/10

	return
}

func population(rand *rand.Rand) (total uint64, base int) {
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
	total = uint64(math.Exp(float64(float32(base) + log) * math.Log(10)))
	return
}

var govByBase = []string{
	"No government",
	"Oompany/Corporation",
	"Participating Democracy",
	"Self-Perpetuating Oligarchy",
	"Representative Democracy",
	"Feudal Technocracy",
	"Captive Government",
	"Balkanizatiomn",
	"Civil Service Bereaucracy",
	"Impersonal Bereaucracy",
	"Charismatic Dictator",
	"Non-Charismatic Leader",
	"Charismatic Oligarchy",
	"Religious Dictatorship",
}

func government(rand *rand.Rand, popBase int) (description string, base int) {
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
	"Most firearsm (except shotgun) prohibited, weapons discouraged",
	"Shotguns prohibited",
	"Long bladed weapons prohibited",
	"Possession of any weappon outside residence prohibited",
}

func lawLevel(rand *rand.Rand, govBase int) (description string, base int) {
	base = d6(rand) + d6(rand) + govBase - 7
	if base < 0 {
		base = 0
	} else if base >= len(lawLevelByBase) {
		base = len(lawLevelByBase) - 1
	}
	description = lawLevelByBase[base]
	return
}

func techLevel(rand *rand.Rand, starPort string, size int, atm int, hydro int, pop int, gov int) (techLevel string, tl int) {
	diceModifier := 0
	if starPort == "A" {
		diceModifier = 6
	} else if starPort == "B" {
		diceModifier = 4
	} else if starPort == "C" {
		diceModifier = 2
	} else if starPort == "X" {
		diceModifier = -4
	}
	if size < 2 {
		diceModifier += 2
	} else if size < 5 {
		diceModifier++
	}
	if atm < 4 || atm > 9 {
		diceModifier++
	}
	if hydro > 8 {
		diceModifier += hydro - 8
	}
	if pop > 0 && pop < 6 {
		diceModifier += 1
	} else if pop > 8 {
		diceModifier += 2 * (pop - 8)
	}
	if gov == 0 || gov == 5 {
		diceModifier++
	} else if gov == 13 {
		diceModifier -= 2
	}
	tl = d6(rand) + diceModifier
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

func twoFloatD6(rand *rand.Rand) (result float32) {
	result = 6.0*rand.Float32() + 6.0*rand.Float32() + 2.0
	if result * (1 + 1/36) >= 12 {
		result = 12.0
	} else {
		result = (result - 2.0) * (1 + 1/36) + 2.0
	}
	return
}

func d6(rand *rand.Rand) (result int) {
	result = rand.Intn(6) + 1
	return
}

package main

import (
	"fmt"
	"github.com/goki/ki/kit"
	"math"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

type selectFunc func() ([]*star)

type systemSelector struct {
	currentSystem int
	scene          *gi3d.Scene
	toolBar        *gi.ToolBar
	jumpComboBox   *gi.ComboBox
	filterComboBox *gi.ComboBox
	viewPort       *gi.Viewport2D
	sceneView      *gi3d.SceneView
	win            *gi.Window
	star           *star
	targets        []int
	choose         selectFunc
}

var selection = systemSelector{
	currentSystem: 0,
	scene:          &gi3d.Scene{},
	toolBar:        &gi.ToolBar{},
	jumpComboBox:   &gi.ComboBox{},
	filterComboBox: &gi.ComboBox{},
	viewPort:       &gi.Viewport2D{},
	sceneView:      &gi3d.SceneView{},
	star:           &star{},
	targets:        []int{},
	choose:         allStars,
}

const hdrText = `<p>Star %d </p>
	<p><b>StarPort</b> %s</p>
	<p><b>Size</b> %d  </p>
	<p><b>Atmosphere</b> %s  </p>
    <p><b>Size</b> %d  </p>
    <p><b>HydroSize</b> %d  </p>
	<p><b>Population</b> %d</p>
    <p>%s</p>
    <p><b>Law Level</b> %d</p>
    <p><b>Tech Level</b> %d</p>
    <p><b>Tech Description</b> %s</p>`

var KiT_SceneView = kit.Types.AddType(&gi3d.SceneView{}, nil)

var (
	filter = map[string] selectFunc{
		"All":             allStars,
		"High Tech":       maxTech,
		"Dry Worlds":      starHydroMin,
		"Water Worlds":    starHydroMax,
		"Largest Worlds":  maxSize,
		"No Worlds":       minSize,
		"Populous Worlds": maxPop,
		"EMPTY Worlds":    minPop,
	}
)

func (s *systemSelector) updateWorldLableTextAndCamera(systemID int) (header string){
		removeSel := s.toolBar.ChildByName("selmode", 0)
		if removeSel != nil{
		s.toolBar.DeleteChild(removeSel, true)
	}

	if s.filterComboBox == nil || s.filterComboBox.Name() != "selFilter"{
		s.filterComboBox = gi.AddNewComboBox(s.toolBar, "selFilter")
	}
	selections := make([]string, 0)
	s.targets = make([]int, 0)
	for key, _ := range filter {
		selections = append(selections, key)
	}

	s.filterComboBox.ItemsFromStringList(selections, true, len(selections))

	s.filterComboBox.SetCurIndex(int(s.sceneView.Scene().SelMode))
	s.filterComboBox.ComboSig.ConnectOnly(s.sceneView.This(), s.filterHandler)

	if s.jumpComboBox == nil || s.jumpComboBox.Name() != "selJump" {
	s.jumpComboBox = gi.AddNewComboBox(s.toolBar, "selJump")
	}
	selections = make([]string, 0)
	s.targets = make([]int, 0)
	for id, jump := range jumpsByStar[systemID] {
		nextStar := -1
		if systemID == jump.s1ID {
			nextStar = jump.s2ID
		} else {
			nextStar = jump.s1ID
		}
		if nextStar != -1 {
			selections = append(selections, fmt.Sprintf("Jump #%d to star %d", id+1, nextStar))
			s.targets = append(s.targets, nextStar)
		}
	}
	s.jumpComboBox.ItemsFromStringList(selections, true, len(selections))

	s.jumpComboBox.SetCurIndex(int(s.sceneView.Scene().SelMode))
	s.jumpComboBox.ComboSig.ConnectOnly(s.sceneView.This(), s.handler)

	s.scene.SetActiveStateUpdt(true)

	s.star = stars[systemID]
	header = worldFromStar(systemID).worldHeader
	workingWorld.SystemDetails.Redrawable = true
	workingWorld.worldHeader = header
	workingWorld.SystemDetails.CurBgColor = gist.Color{R: 0, G: 0, B: 0, A: 255}
	workingWorld.SystemDetails.SetText(header)
	s.scene.Camera.Pose.Pos.Set(float32(stars[systemID].x) + offsets.x, float32(stars[systemID].y) + offsets.y, float32(stars[systemID].z+offsets.z) + 0.1)
	s.scene.Camera.LookAt(mat32.Vec3{
		X: float32(stars[systemID].x) + offsets.x,
		Y: float32(stars[systemID].y) + offsets.y,
		Z: float32(stars[systemID].z) + offsets.z,
	}, mat32.Vec3{
		X: 0,
		Y: .1,
		Z: 0,
	})

	s.scene.SetActiveStateUpdt(false)
	for id, l := range lines {
		thicker := float32(1.0)
		if l.jumpInfo.s1ID == systemID ||
			l.jumpInfo.s2ID == systemID {
			l.jumpInfo.activeColor.R = l.jumpInfo.color.R + eighth
			l.jumpInfo.color.G = l.jumpInfo.color.G + eighth
			l.jumpInfo.color.B = l.jumpInfo.color.B + eighth
			thicker = float32(10.0)
		}
		thickness := float32(0.00005)
		if l.jumpInfo.color.A < math.MaxUint8-55 {
			thickness = 0.00010 * thicker
		} else if l.jumpInfo.color.A < math.MaxUint8-47 {
			thickness = 0.00012 * thicker
		} else if l.jumpInfo.color.A < math.MaxUint8-39 {
			thickness = 0.00015 * thicker
		}
		lines[id].lines.Width = mat32.Vec2{X: thickness, Y: thickness}
	}
	return
}

func (s *systemSelector) handler(recv, send ki.Ki, sig int64, data interface{}) {
	svv := recv.Embed(KiT_SceneView).(*gi3d.SceneView)
	cbb := send.(*gi.ComboBox)
	//scc := svv.Scene()
	if cbb.CurIndex < len(s.targets) {
		s.currentSystem = s.targets[cbb.CurIndex]
		s.updateWorldLableTextAndCamera(s.targets[cbb.CurIndex])
		svv.UpdateSig()
	}
}

func (s *systemSelector) filterHandler(recv, send ki.Ki, sig int64, data interface{}) {
	svv := recv.Embed(KiT_SceneView).(*gi3d.SceneView)
	cbb := send.(*gi.ComboBox)
	//scc := svv.Scene()
	if cbb.CurIndex < len(filter) {
		sel := cbb.CurVal.(string)
		if filter[sel] != nil {
			s.choose = filter[sel]
		}
		s.currentSystem = s.choose()[0].id
		s.updateWorldLableTextAndCamera(s.currentSystem)
		svv.UpdateSig()
	}
}
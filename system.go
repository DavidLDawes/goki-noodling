package main

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

type systemSelector struct {
	currentSystem int
	scene         *gi3d.Scene
	toolBar       *gi.ToolBar
	comboBox      *gi.ComboBox
	viewPort      *gi.Viewport2D
	sceneView     *gi3d.SceneView
	win           *gi.Window
	star          *star
}

var selection = systemSelector{
	currentSystem: 0,
	scene:         &gi3d.Scene{},
	toolBar:       &gi.ToolBar{},
	comboBox:      &gi.ComboBox{},
	viewPort:      &gi.Viewport2D{},
	sceneView:     &gi3d.SceneView{},
	star:          &star{},
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

func (s *systemSelector) updateWorldLableText(systemID int) (header string) {
	removeSel := s.toolBar.ChildByName("selmode", 0)
	if removeSel != nil {
		s.toolBar.DeleteChild(removeSel, true)
	}
	if s.comboBox == nil || s.comboBox.Name() != "selJump" {
		s.comboBox = gi.AddNewComboBox(s.toolBar, "selJump")
	}
	selections := make([]string, 0)
	targets := make([]int, 0)
	for id, jump := range jumpsByStar[systemID] {
		var nextStar int
		if systemID == jump.s1ID {
			nextStar = jump.s2ID
		} else {
			nextStar = jump.s1ID
		}
		selections = append(selections, fmt.Sprintf("Jump #%d to star %d", id+1, nextStar))
		targets = append(targets, nextStar)
	}
	s.comboBox.ItemsFromStringList(selections, true, len(selections))

	s.comboBox.SetCurIndex(int(s.sceneView.Scene().SelMode))
	s.comboBox.ComboSig.ConnectOnly(s.sceneView.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		cbb := send.(*gi.ComboBox)
		s.currentSystem = targets[cbb.CurIndex]
		s.updateWorldLableText(targets[cbb.CurIndex])
		s.scene.UpdateSig()
	})

	s.scene.SetActiveStateUpdt(true)

	s.star = stars[systemID]
	header = worldFromStar(systemID).worldHeader
	workingWorld.SystemDetails.Redrawable = true
	workingWorld.worldHeader = header
	workingWorld.SystemDetails.CurBgColor = gist.Color{R: 0, G: 0, B: 0, A: 255}
	workingWorld.SystemDetails.SetText(header)
	s.scene.Camera.Pose.Pos.Set(float32(stars[systemID].x), float32(stars[systemID].y), float32(stars[systemID].z+offsets.z+.1))
	s.scene.Camera.LookAt(mat32.Vec3{
		X: float32(stars[systemID].x) + offsets.x,
		Y: float32(stars[systemID].y) + offsets.y,
		Z: float32(stars[systemID].z) + offsets.z,
	}, mat32.Vec3{
		X: 0,
		Y: 1,
		Z: 0,
	})
	s.scene.SetActiveStateUpdt(false)

	return
}

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

const (
	width  = 1280
	height = 1024
)

func main() {
	gimain.Main(func() {
		mainRun()
	})
}

func mainRun() {
	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.WinEventTrace = true
	// gi3d.Update3DTrace = true
	// gi.Update2DTrace = true

	win := mainMenuGetReceiver()

	vp := win.WinViewport2D()
	update := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewEx(1))

	outer := getOuter(mfr)
	info := getInfo(mfr, win)

	main3d := gi.AddNewLayout(outer, "main3d", gi.LayoutHoriz)
	main3d.SetStretchMax()

	trow := gi.AddNewLayout(info, "trow", gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	sc := addScene(info)
	renderStars(sc)

	selection.win = win
	selection.scene = sc
	selection.viewPort = vp
	selection.updateWorldLableText(0)
	appName := gi.AppName()
	mainMenu := win.MainMenu
	mainMenu.ConfigMenus([]string{appName, "File", "Edit", "Window"})

	//	amen := win.MainMenu.ChildByName(appName, 0).(*gi.Action)
	//	amen.Menu.AddAppMenu(win)
	win.MainMenuUpdated()

	vp.UpdateEndNoSig(update)
	win.StartEventLoop()
}

func mainMenuGetReceiver() (win *gi.Window) {
	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("galaxy3d")
	gi.SetAppAbout(`Playing with a procedurally generated galaxy. Teaching myself UI in Go using <b>GoKi</b>.` +
		` See <a href="https://github.com/DavidLDawes/goki-noodling"></a>.`)
	win = gi.NewMainWindow("Galaxy 3d", "Dave's 3D Galaxy Demo", width, height)

	return
}

func getOuter(mfr ki.Ki) (outer *gi.Layout) {
	outer = gi.AddNewLayout(mfr, "outer", gi.LayoutHoriz)
	outer.SetStretchMax()

	return
}

type systemSelector struct {
	currentSystem int
	scene         *gi3d.Scene
	viewPort      *gi.Viewport2D
	win           *gi.Window
	star          *star
}

var selection = systemSelector{
	currentSystem: 0,
	scene:         &gi3d.Scene{},
	viewPort:      &gi.Viewport2D{},
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
	s.scene.SetActiveStateUpdt(true)

	s.star = stars[systemID]
	header = worldFromStar(systemID).worldHeader
	workingWorld.SystemDetails.Redrawable = true
	workingWorld.worldHeader = header
	workingWorld.SystemDetails.CurBgColor = gist.Color{R: 0, G: 0, B: 0, A: 255}
	workingWorld.SystemDetails.SetText(header)
	s.scene.Camera.Pose.Pos.Set(float32(stars[systemID].x), float32(stars[systemID].y), float32(stars[systemID].z + .5))
	s.scene.Camera.LookAt(mat32.Vec3{
		X: float32(stars[systemID].x),
		Y: float32(stars[systemID].y),
		Z: float32(stars[systemID].z),
	}, mat32.Vec3Y) // defaults to looking at origin
	s.scene.SetActiveStateUpdt(false)

	return
}

func getInfo(mfr ki.Ki, win *gi.Window) (info *gi.Layout) {
	info = gi.AddNewLayout(mfr, "info", gi.LayoutHoriz)
	info.SetStretchMaxHeight()
	inner := gi.AddNewLayout(mfr, "info", gi.LayoutVert)
	inner.SetStretchMaxHeight()

	putWorldHeader(info)
	addJumpButtons()

	return
}

func addScene(info ki.Ki) (result *gi3d.Scene) {
	//
	//    Scene

	gi.AddNewSpace(info, "screenSpace")
	sceneRow := gi.AddNewLayout(info, "sceneRow", gi.LayoutHoriz)
	sceneRow.SetStretchMax()

	sceneView := gi3d.AddNewSceneView(sceneRow, "sceneView")
	sceneView.SetStretchMax()
	sceneView.Config()
	tbar := sceneView.Toolbar()
	tbar.AddSeparator("select")
	gi.AddNewLabel(tbar, "select", "Select:")
	tbar.AddAction(gi.ActOpts{Icon: "wedge-left"}, sceneView.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			selection.currentSystem--
			if selection.currentSystem < 0 {
				selection.currentSystem = len(stars) - 1
			}
			selection.updateWorldLableText(selection.currentSystem)
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-right"}, sceneView.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			selection.currentSystem++
			if selection.currentSystem > len(stars) - 1 {
				selection.currentSystem = 0
			}
			selection.updateWorldLableText(selection.currentSystem)
		})
	result = sceneView.Scene()
	// sc.NoNav = true

	// first, add lights, set camera
	result.BgColor.SetUInt8(0, 0, 0, 255) // sky blue-ish
	gi3d.AddNewAmbientLight(result, "ambient", 0.6, gi3d.DirectSun)

	return
}

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

const (
	width  = 1440
	height = 1280
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
	selection.updateWorldLableTextAndCamera(0)
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

func getInfo(mfr ki.Ki, win *gi.Window) (info *gi.Layout) {
	info = gi.AddNewLayout(mfr, "info", gi.LayoutHoriz)
	info.SetStretchMaxHeight()
	inner := gi.AddNewLayout(mfr, "info", gi.LayoutVert)
	inner.SetStretchMaxHeight()

	putWorldHeader(info)

	return
}

func addScene(info ki.Ki) (result *gi3d.Scene) {
	//
	//    Scene

	gi.AddNewSpace(info, "screenSpace")
	sceneRow := gi.AddNewLayout(info, "sceneRow", gi.LayoutHoriz)
	sceneRow.SetStretchMax()

	sceneView := gi3d.AddNewSceneView(sceneRow, "sceneView")
	selection.sceneView = sceneView
	sceneView.SetStretchMax()
	sceneView.Config()
	tbar := sceneView.Toolbar()
	selection.toolBar = tbar

	removeSel := tbar.ChildByName("selmode", 0)
	if removeSel != nil {
		tbar.DeleteChild(removeSel, false)
	}
	removeEdit := tbar.ChildByName("Edit", 0)
	if removeEdit != nil {
		tbar.DeleteChild(removeEdit, false)
	}
	removeEdit = tbar.ChildByName("Edit Scene", 0)
	if removeEdit != nil {
		tbar.DeleteChild(removeEdit, false)
	}
	gi.AddNewLabel(tbar, "select", "Select:")
	tbar.AddAction(gi.ActOpts{Icon: "wedge-left"}, sceneView.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			selection.currentSystem--
			if selection.currentSystem < 0 {
				selection.currentSystem = len(selection.choose()) - 1
			}
			selection.updateWorldLableTextAndCamera(selection.currentSystem)
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-right"}, sceneView.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			selection.currentSystem++
			if selection.currentSystem > len(selection.choose())-1 {
				selection.currentSystem = 0
			}
			selection.updateWorldLableTextAndCamera(selection.currentSystem)
		})
	result = sceneView.Scene()
	result.BgColor.SetUInt8(0, 0, 0, 255)
	gi3d.AddNewAmbientLight(result, "ambient", 0.6, gi3d.DirectSun)

	return
}

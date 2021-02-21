package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.WinEventTrace = true
	// gi3d.Update3DTrace = true
	// gi.Update2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("galaxy3d")
	gi.SetAppAbout(`Playing with a procedurally generated galaxy. Teaching myself UI in Go using <b>GoKi</b>. See <a href="https://github.com/DavidLDawes/goki-noodling"></a>.`)

	win := gi.NewMainWindow("Galaxy 3d", "Dave's 3D Galaxy Demo", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewEx(1))

	trow := gi.AddNewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	//////////////////////////////////////////
	//    Scene

	gi.AddNewSpace(mfr, "scspc")
	scrow := gi.AddNewLayout(mfr, "scrow", gi.LayoutHoriz)
	scrow.SetStretchMax()

	scvw := gi3d.AddNewSceneView(scrow, "sceneview")
	scvw.SetStretchMax()
	scvw.Config()
	sc := scvw.Scene()
	// sc.NoNav = true

	// first, add lights, set camera
	sc.BgColor.SetUInt8(0, 0, 0, 255) // sky blue-ish

	gi3d.AddNewAmbientLight(sc, "ambient", 0.6, gi3d.DirectSun)

	renderStars(sc)

	sc.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin
	//	menu config etc

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	//	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	//	amen.Menu.AddAppMenu(win)
	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}

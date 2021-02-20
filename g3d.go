package main

import (
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

// Anim has control for animating
type Anim struct {
	On     bool         `desc:"run the animation"`
	Speed  float32      `min:"0.01" step:"0.01" desc:"angular speed (in radians)"`
	Ang    float32      `inactive:"+" desc:"current angle"`
	Ticker *time.Ticker `view:"-" desc:"the time.Ticker for animating the scene"`
	Scene  *gi3d.Scene  `desc:"the scene"`
}

// Start starts the animation ticker timer -- if on is true, then
// animation will actually start too.
func (an *Anim) Start(sc *gi3d.Scene, on bool) {
	an.Scene = sc
	an.On = on
	an.Speed = .1
	an.GetObjs()
}

// GetObjs gets the objects to animate
func (an *Anim) GetObjs() {
	ggp := an.Scene.ChildByName("go-group", 0)
	if ggp == nil {
		return
	}
	gophi := ggp.Child(1)
	if gophi == nil {
		return
	}
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

	gi.SetAppName("gi3d")
	gi.SetAppAbout(`This is a demo of the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/gi3d/README.md">README</a> page for this example app has further info.</p>`)

	win := gi.NewMainWindow("gogi-gi3d-demo", "GoGi 3D Demo", width, height)

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

	// gi.AddNewLabel(scrow, "tmp", "This is test text")

	scvw := gi3d.AddNewSceneView(scrow, "sceneview")
	scvw.SetStretchMax()
	scvw.Config()
	sc := scvw.Scene()
	// sc.NoNav = true

	// first, add lights, set camera
	sc.BgColor.SetUInt8(0, 0, 0, 255) // sky blue-ish

	gi3d.AddNewAmbientLight(sc, "ambient", 0.6, gi3d.DirectSun)

	renderStars(sc)

	txt := gi3d.AddNewText2D(sc, sc, "text", "Text2D can put <b>HTML</b> formatted<br>Text anywhere you might <i>want</i>")
	// 	txt.SetProp("background-color", gist.Color{0, 0, 0, 0}) // transparent -- default
	// txt.SetProp("background-color", "white")
	txt.SetProp("color", "black") // default depends on Light / Dark mode, so we set this
	// txt.SetProp("margin", units.NewPt(4)) // default is 2 px
	// txt.Mat.Bright = 5 // no dim text -- key if using a background and want it to be bright..
	txt.SetProp("text-align", gist.AlignLeft) // gi.AlignCenter)
	txt.Pose.Scale.SetScalar(0.2)
	txt.Pose.Pos.Set(0, 2.2, 0)

	sc.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	///////////////////////////////////////////////////
	//  Animation & Embedded controls

	anim := &Anim{}
	anim.Start(sc, false) // start without animation running

	emb := gi3d.AddNewEmbed2D(sc, sc, "embed-but", 150, 100, gi3d.FitContent)
	emb.Pose.Pos.Set(-2, 2, 0)
	// emb.Zoom = 1.5   // this is how to rescale overall size
	evlay := gi.AddNewFrame(emb.Viewport, "vlay", gi.LayoutVert)
	evlay.SetProp("margin", units.NewEx(1))

	eabut := gi.AddNewCheckBox(evlay, "anim-but")
	eabut.SetText("Animate")
	eabut.Tooltip = "toggle animation on and off"
	eabut.ButtonSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			anim.On = !eabut.IsChecked()
		}
	})

	cmb := gi.AddNewMenuButton(evlay, "anim-ctrl")
	cmb.SetText("Anim Ctrl")
	cmb.Tooltip = "options for what is animated (note: menu only works when not animating -- checkboxes would be more useful here but wanted to test menu function)"
	cmb.Menu.AddAction(gi.ActOpts{Label: "Edit Anim"},
		win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			giv.StructViewDialog(vp, anim, giv.DlgOpts{Title: "Animation Parameters"}, nil, nil)
		})

	sprw := gi.AddNewLayout(evlay, "speed-lay", gi.LayoutHoriz)
	gi.AddNewLabel(sprw, "speed-lbl", "Speed: ")
	sb := gi.AddNewSpinBox(sprw, "anim-speed")
	sb.Defaults()
	sb.HasMin = true
	sb.Min = 0.01
	sb.Step = 0.01
	sb.SetValue(anim.Speed)
	sb.Tooltip = "determines the speed of rotation (step size)"

	spsld := gi.AddNewSlider(evlay, "speed-slider")
	spsld.Dim = mat32.X
	spsld.Defaults()
	spsld.Min = 0.01
	spsld.Max = 1
	spsld.Step = 0.01
	spsld.PageStep = 0.1
	spsld.SetMinPrefWidth(units.NewEm(20))
	spsld.SetMinPrefHeight(units.NewEm(2))
	spsld.SetValue(anim.Speed)
	// spsld.Tracking = true
	spsld.Icon = gi.IconName("circlebutton-on")

	sb.SpinBoxSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		anim.Speed = sb.Value
		spsld.SetValue(anim.Speed)
	})
	spsld.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if gi.SliderSignals(sig) == gi.SliderValueChanged {
			anim.Speed = data.(float32)
			sb.SetValue(anim.Speed)
		}
	})

	//	menu config etc

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)
	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}

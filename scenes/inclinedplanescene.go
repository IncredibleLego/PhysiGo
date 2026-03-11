package scenes

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"physiGo/config"
	"physiGo/utils"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type inclinedPlaneSnapshot struct {
	theta      float64
	muS        float64
	muK        float64
	mass       float64
	gravity    float64
	length     float64
	hBlock     float64
	v0         float64
	muSSet     bool
	muKSet     bool
	gravitySet bool
}

type InclinedPlaneScene struct {
	theta   float64
	muS     float64
	muK     float64
	mass    float64
	gravity float64
	length  float64
	hBlock  float64
	v0      float64

	muSSet     bool
	muKSet     bool
	gravitySet bool

	calc        InclinedPlaneCalculus
	snapshot    inclinedPlaneSnapshot
	started     bool
	completed   bool
	simTime     float64
	simS        float64
	simVelocity float64
	simHBlock   float64
	simHorizS   float64
	phase       string
	tPhaseStart float64

	baseReached         bool
	baseReachTime       float64
	baseReachVelocity   float64
	baseReachDistance   float64
	simulationEnded     bool
	simulationEndTime   float64
	simulationEndHorizS float64

	playImage  *ebiten.Image
	pauseImage *ebiten.Image
	planeImage *ebiten.Image
	blockImage *ebiten.Image
	whiteImage *ebiten.Image
}

func (i *InclinedPlaneScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return reason == Unpause
}

func NewInclinedPlaneScene() *InclinedPlaneScene {
	return &InclinedPlaneScene{}
}

func (i *InclinedPlaneScene) Draw(screen *ebiten.Image) {
	screen.Clear()

	textDim := config.GlobalConfig.TextDimension
	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	leftX := sw * 0.03
	rightX := sw * 0.80
	y := textDim * 0.75
	step := textDim * 0.68
	smallSize := -(textDim * 0.54)

	status := "REST - Press SPACE to start"
	if i.started && !i.completed {
		status = "RUNNING - Press SPACE to pause"
	}
	if i.completed {
		status = "COMPLETED - Press R to reset"
	}

	leftLines := []string{
		"LIVE DATA",
		fmt.Sprintf("status: %s", status),
		fmt.Sprintf("phase: %s", i.phase),
		fmt.Sprintf("t: %.2f s", i.simTime),
		fmt.Sprintf("a: %.2f m/s^2", i.currentAcceleration()),
		fmt.Sprintf("v: %.2f m/s", i.simVelocity),
		fmt.Sprintf("s (incline): %.2f m", i.simS),
		fmt.Sprintf("x (ground): %.2f m", i.simHorizS),
		fmt.Sprintf("h block: %.2f m", i.simHBlock),
		fmt.Sprintf("dist from base: %.2f m", i.distanceFromBase()),
		fmt.Sprintf("dist from origin: %.2f m", i.simS+i.simHorizS),
	}

	rightLines := []string{
		"PLANE DATA",
		fmt.Sprintf("mass: %.1f kg", i.mass),
		fmt.Sprintf("gravity: %.1f m/s^2", i.gravity),
		fmt.Sprintf("θ: %.1f°", i.theta),
		fmt.Sprintf("length L: %.1f m", i.length),
		fmt.Sprintf("h0 block: %.1f m", i.initialHeight()),
		fmt.Sprintf("v0: %.1f m/s", i.v0),
		fmt.Sprintf("μ_s: %.2f", i.muS),
		fmt.Sprintf("μ_k: %.2f", i.muK),
		fmt.Sprintf("P||: %.1f N", i.calc.WeightParallel),
		fmt.Sprintf("N: %.1f N", i.calc.Normal),
		fmt.Sprintf("Fs,max: %.1f N", i.calc.StaticFrictionMax),
		fmt.Sprintf("Fk: %.1f N", i.calc.DynamicFriction),
		fmt.Sprintf("F net: %.1f N", i.calc.NetForce),
		fmt.Sprintf("slides: %t", i.calc.Slides),
	}

	liveDataBottom := y + float64(len(leftLines))*step + textDim*0.25
	i.drawInclinedPlane(screen, liveDataBottom)

	for idx, line := range leftLines {
		col := "white"
		if idx == 0 {
			col = "yellow"
		}
		if idx == 1 {
			col = "cyan"
		}
		utils.ScreenDraw(smallSize, leftX, y+float64(idx)*step, col, screen, line, "libertinus")
	}

	for idx, line := range rightLines {
		col := "white"
		if idx == 0 {
			col = "yellow"
		}
		utils.ScreenDraw(smallSize, rightX, y+float64(idx)*step, col, screen, line, "libertinus")
	}

	resultY := textDim * 0.78
	resultGap := step * 0.9
	textSize := textDim + smallSize
	if textSize < 8 {
		textSize = 8
	}
	drawCentered := func(line, col string, yy float64) {
		w, _ := utils.MeasureTextWithSize(line, textSize, "libertinus")
		x := sw/2 - w/2
		utils.ScreenDraw(smallSize, x, yy, col, screen, line, "libertinus")
	}

	resultLines := make([]struct {
		text string
		col  string
	}, 0, 3)
	if i.baseReached {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: fmt.Sprintf("BASE t=%.2fs  v=%.2fm/s", i.baseReachTime, i.baseReachVelocity),
			col:  "green",
		})
	}
	if i.simulationEnded {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: fmt.Sprintf("STOP t=%.2fs  x=%.2fm", i.simulationEndTime, i.simulationEndHorizS),
			col:  "green",
		})
	}
	if i.baseReached && !i.simulationEnded && i.calc.HorizontalDecel <= 0 {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: "No horizontal friction",
			col:  "orange",
		})
	}
	if !i.calc.Slides {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: "BLOCK DOES NOT MOVE",
			col:  "red",
		})
	}

	for idx, line := range resultLines {
		drawCentered(line.text, line.col, resultY+float64(idx)*resultGap)
	}

	controls := "SPACE: start/pause  <-/->: scrub timeline  R: reset  ENTER: menu"
	utils.ScreenDraw(smallSize, textDim/3, sh-textDim*0.9, "light gray", screen, controls, "libertinus")

	i.drawTimelineControls(screen)
}

func (i *InclinedPlaneScene) FirstLoad() {
	i.theta = config.GlobalConfig.InclinedTheta
	i.muS = config.GlobalConfig.InclinedMuS
	i.muK = config.GlobalConfig.InclinedMuK
	i.mass = config.GlobalConfig.InclinedMass
	i.gravity = config.GlobalConfig.InclinedGravity
	i.length = config.GlobalConfig.InclinedLength
	i.hBlock = config.GlobalConfig.InclinedHBlock
	i.v0 = config.GlobalConfig.InclinedInitialVelocity
	i.muSSet = config.GlobalConfig.InclinedMuSSet
	i.muKSet = config.GlobalConfig.InclinedMuKSet
	i.gravitySet = config.GlobalConfig.InclinedGravitySet

	i.snapshot = inclinedPlaneSnapshot{
		theta:      i.theta,
		muS:        i.muS,
		muK:        i.muK,
		mass:       i.mass,
		gravity:    i.gravity,
		length:     i.length,
		hBlock:     i.hBlock,
		v0:         i.v0,
		muSSet:     i.muSSet,
		muKSet:     i.muKSet,
		gravitySet: i.gravitySet,
	}

	i.refreshCalculus()
	i.resetSimulationFromSnapshot()
	i.loadControlImages()
}

func (i *InclinedPlaneScene) OnEnter() {
}

func (i *InclinedPlaneScene) OnExit() {
}

func (i *InclinedPlaneScene) Update() SceneId {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		i.toggleRunState()
	}

	i.handleKeyboardScrub()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		i.handleMouseControl()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		i.resetSimulationFromSnapshot()
	}

	i.stepSimulation()

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return PauseSceneId
	}

	return InclinedPlaneSceneId
}

func (i *InclinedPlaneScene) handleKeyboardScrub() {
	total := i.totalSimulationDuration()
	if total <= 0 {
		return
	}

	step := total * 0.02
	if step < 0.05 {
		step = 0.05
	}

	advance := func(key ebiten.Key, delta float64) {
		pressed := inpututil.IsKeyJustPressed(key)
		dur := inpututil.KeyPressDuration(key)
		if !pressed && dur > 10 && dur%3 == 0 {
			pressed = true
		}
		if !pressed {
			return
		}
		i.started = false
		i.completed = false
		i.setSimulationTime(i.simTime + delta)
	}

	advance(ebiten.KeyArrowRight, step)
	advance(ebiten.KeyArrowLeft, -step)
}

func (i *InclinedPlaneScene) refreshCalculus() {
	i.calc = ComputeInclinedPlaneCalculus(&config.Config{
		InclinedTheta:           i.theta,
		InclinedMuS:             i.muS,
		InclinedMuK:             i.muK,
		InclinedMass:            i.mass,
		InclinedGravity:         i.gravity,
		InclinedLength:          i.length,
		InclinedHBlock:          i.hBlock,
		InclinedInitialVelocity: i.v0,
		InclinedMuSSet:          i.muSSet,
		InclinedMuKSet:          i.muKSet,
		InclinedGravitySet:      i.gravitySet,
	})
}

func (i *InclinedPlaneScene) resetSimulationFromSnapshot() {
	i.theta = i.snapshot.theta
	i.muS = i.snapshot.muS
	i.muK = i.snapshot.muK
	i.mass = i.snapshot.mass
	i.gravity = i.snapshot.gravity
	i.length = i.snapshot.length
	i.hBlock = i.snapshot.hBlock
	i.v0 = i.snapshot.v0
	i.muSSet = i.snapshot.muSSet
	i.muKSet = i.snapshot.muKSet
	i.gravitySet = i.snapshot.gravitySet

	i.started = false
	i.completed = false
	i.simTime = 0
	i.simS = 0
	i.simVelocity = i.v0
	i.simHBlock = i.initialHeight()
	i.simHorizS = 0
	i.phase = "ready"
	i.tPhaseStart = 0
	i.baseReached = false
	i.baseReachTime = 0
	i.baseReachVelocity = 0
	i.baseReachDistance = 0
	i.simulationEnded = false
	i.simulationEndTime = 0
	i.simulationEndHorizS = 0

	i.refreshCalculus()
}

func (i *InclinedPlaneScene) stepSimulation() {
	if !i.started || i.completed || !i.calc.Slides {
		return
	}

	tps := ebiten.ActualTPS()
	if tps <= 0 {
		tps = 60
	}
	dt := 1.0 / tps

	i.setSimulationTime(i.simTime + dt)
}

func (i *InclinedPlaneScene) totalSimulationDuration() float64 {
	if !i.calc.Slides {
		return 0
	}
	if i.calc.StopsOnIncline {
		return i.calc.TimeToBase
	}
	if i.calc.HorizontalDecel > 0 {
		return i.calc.TimeToBase + i.calc.HorizontalStopTime
	}
	return i.calc.TimeToBase
}

func (i *InclinedPlaneScene) simulationProgress() float64 {
	total := i.totalSimulationDuration()
	if total <= 0 {
		if i.completed {
			return 1
		}
		return 0
	}
	p := i.simTime / total
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func (i *InclinedPlaneScene) baseReachProgress() float64 {
	if !i.calc.Slides || i.calc.StopsOnIncline || i.calc.TimeToBase <= 0 {
		return -1
	}
	total := i.totalSimulationDuration()
	if total <= 0 {
		return -1
	}
	p := i.calc.TimeToBase / total
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func (i *InclinedPlaneScene) setSimulationTime(t float64) {
	if !i.calc.Slides {
		i.simTime = 0
		i.simS = 0
		i.simHorizS = 0
		i.simVelocity = i.calc.InitialVelocity
		i.simHBlock = i.initialHeight()
		i.phase = "ready"
		i.completed = false
		i.baseReached = false
		i.simulationEnded = false
		i.started = false
		return
	}

	if t < 0 {
		t = 0
	}

	total := i.totalSimulationDuration()
	if total > 0 && i.calc.HorizontalDecel > 0 && t > total {
		t = total
	}

	v0 := i.calc.InitialVelocity
	a := i.calc.Acceleration

	i.simTime = t
	i.simS = 0
	i.simHorizS = 0
	i.simVelocity = v0
	i.simHBlock = i.initialHeight()
	i.phase = "ready"
	i.tPhaseStart = 0
	i.completed = false
	i.baseReached = false
	i.baseReachTime = 0
	i.baseReachVelocity = 0
	i.baseReachDistance = 0
	i.simulationEnded = false
	i.simulationEndTime = 0
	i.simulationEndHorizS = 0

	if t == 0 {
		return
	}

	inclineTime := i.calc.TimeToBase

	if i.calc.StopsOnIncline {
		tIncline := t
		if tIncline > inclineTime {
			tIncline = inclineTime
		}

		i.phase = "incline"
		i.simS = v0*tIncline + 0.5*a*tIncline*tIncline
		i.simVelocity = v0 + a*tIncline
		if i.simVelocity < 0 {
			i.simVelocity = 0
		}
		if i.simS < 0 {
			i.simS = 0
		}

		thetaRad := i.theta * math.Pi / 180.0
		heightLost := i.simS * math.Sin(thetaRad)
		i.simHBlock = i.initialHeight() - heightLost
		if i.simHBlock < 0 {
			i.simHBlock = 0
		}

		if t >= inclineTime {
			i.simS = i.calc.StopDistanceOnIncline
			i.simVelocity = 0
			i.phase = "stopped"
			i.completed = true
			i.started = false
			i.simulationEnded = true
			i.simulationEndTime = inclineTime
			i.simulationEndHorizS = 0
			i.simTime = inclineTime
		}
		return
	}

	if t < inclineTime {
		i.phase = "incline"
		i.simS = v0*t + 0.5*a*t*t
		i.simVelocity = v0 + a*t
		if i.simVelocity < 0 {
			i.simVelocity = 0
		}
		if i.simS < 0 {
			i.simS = 0
		}

		thetaRad := i.theta * math.Pi / 180.0
		heightLost := i.simS * math.Sin(thetaRad)
		i.simHBlock = i.initialHeight() - heightLost
		if i.simHBlock < 0 {
			i.simHBlock = 0
		}
		return
	}

	i.phase = "horizontal"
	i.simS = i.calc.DistanceToBase
	i.simHBlock = 0
	i.baseReached = true
	i.baseReachTime = inclineTime
	i.baseReachVelocity = i.calc.VelocityAtBase
	i.baseReachDistance = i.simS

	horizontalT := t - inclineTime
	if horizontalT < 0 {
		horizontalT = 0
	}

	if i.calc.HorizontalDecel <= 0 {
		i.simHorizS = i.calc.VelocityAtBase * horizontalT
		i.simVelocity = i.calc.VelocityAtBase
		return
	}

	vBase := i.calc.VelocityAtBase
	decel := i.calc.HorizontalDecel
	i.simHorizS = vBase*horizontalT - 0.5*decel*horizontalT*horizontalT
	i.simVelocity = vBase - decel*horizontalT

	if i.simVelocity <= 0 || horizontalT >= i.calc.HorizontalStopTime {
		i.simVelocity = 0
		i.simHorizS = i.calc.HorizontalStopDist
		i.phase = "stopped"
		i.completed = true
		i.started = false
		i.simulationEnded = true
		i.simulationEndTime = inclineTime + i.calc.HorizontalStopTime
		i.simulationEndHorizS = i.simHorizS
		i.simTime = i.simulationEndTime
	}
}

func (i *InclinedPlaneScene) toggleRunState() {
	if !i.calc.Slides {
		return
	}
	if i.completed {
		i.setSimulationTime(0)
	}
	i.started = !i.started
}

func (i *InclinedPlaneScene) loadControlImages() {
	load := func(path string) *ebiten.Image {
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		img, _, err := image.Decode(file)
		if err != nil {
			return nil
		}
		return ebiten.NewImageFromImage(img)
	}

	i.playImage = load("img/play.png")
	i.pauseImage = load("img/pause.png")
	i.planeImage = load("img/plane.jpg")
	i.blockImage = load("img/block.png")
	i.whiteImage = ebiten.NewImage(1, 1)
	i.whiteImage.Fill(color.White)
}

func (i *InclinedPlaneScene) playButtonRect() (float64, float64, float64, float64) {
	textDim := config.GlobalConfig.TextDimension
	buttonSize := textDim * 1.25
	barX := float64(config.GlobalConfig.ScreenWidth) * 0.2
	barY := float64(config.GlobalConfig.ScreenHeight) - textDim*2.1
	buttonX := barX - buttonSize - textDim*0.45
	buttonY := barY - (buttonSize-textDim*0.5)/2
	return buttonX, buttonY, buttonSize, buttonSize
}

func (i *InclinedPlaneScene) progressBarRect() (float64, float64, float64, float64) {
	textDim := config.GlobalConfig.TextDimension
	barX := float64(config.GlobalConfig.ScreenWidth) * 0.2
	barW := float64(config.GlobalConfig.ScreenWidth) * 0.6
	barH := textDim * 0.45
	barY := float64(config.GlobalConfig.ScreenHeight) - textDim*2.0
	return barX, barY, barW, barH
}

func (i *InclinedPlaneScene) drawTimelineControls(screen *ebiten.Image) {
	btnX, btnY, btnW, btnH := i.playButtonRect()
	barX, barY, barW, barH := i.progressBarRect()

	vector.DrawFilledRect(screen, float32(btnX), float32(btnY), float32(btnW), float32(btnH), color.RGBA{40, 40, 40, 255}, false)

	icon := i.playImage
	if i.started && !i.completed {
		icon = i.pauseImage
	}

	if icon != nil {
		imgW, imgH := icon.Bounds().Dx(), icon.Bounds().Dy()
		if imgW > 0 && imgH > 0 {
			scale := math.Min(btnW*0.8/float64(imgW), btnH*0.8/float64(imgH))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scale, scale)
			drawW := float64(imgW) * scale
			drawH := float64(imgH) * scale
			op.GeoM.Translate(btnX+(btnW-drawW)/2, btnY+(btnH-drawH)/2)
			screen.DrawImage(icon, op)
		}
	} else {
		label := "PLAY"
		if i.started && !i.completed {
			label = "PAUSE"
		}
		x := btnX + btnW*0.12
		y := btnY + btnH*0.68
		utils.ScreenDraw(-(config.GlobalConfig.TextDimension / 2.2), x, y, "white", screen, label, "libertinus")
	}

	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW), float32(barH), color.RGBA{55, 55, 55, 255}, false)
	progress := i.simulationProgress()
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW*progress), float32(barH), color.RGBA{30, 170, 90, 255}, false)

	baseProgress := i.baseReachProgress()
	if baseProgress >= 0 {
		baseX := barX + barW*baseProgress
		vector.DrawFilledRect(screen, float32(baseX-2), float32(barY-3), 4, float32(barH+6), color.RGBA{240, 95, 40, 255}, false)
	}

	knobX := barX + barW*progress
	if knobX < barX {
		knobX = barX
	}
	if knobX > barX+barW {
		knobX = barX + barW
	}
	vector.DrawFilledRect(screen, float32(knobX-2), float32(barY-4), 4, float32(barH+8), color.RGBA{230, 230, 230, 255}, false)
}

func (i *InclinedPlaneScene) handleMouseControl() {
	mx, my := ebiten.CursorPosition()
	px := float64(mx)
	py := float64(my)

	btnX, btnY, btnW, btnH := i.playButtonRect()
	if px >= btnX && px <= btnX+btnW && py >= btnY && py <= btnY+btnH {
		i.toggleRunState()
		return
	}

	barX, barY, barW, barH := i.progressBarRect()
	if px >= barX && px <= barX+barW && py >= barY && py <= barY+barH {
		total := i.totalSimulationDuration()
		if total <= 0 {
			return
		}
		wasRunning := i.started
		progress := (px - barX) / barW
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		i.setSimulationTime(total * progress)
		i.started = wasRunning && !i.completed
	}
}

func (i *InclinedPlaneScene) estimatedTimeToGround() float64 {
	if !i.calc.Slides || i.initialHeight() <= 0 {
		return -1
	}
	if i.calc.StopsOnIncline {
		return -1
	}
	return i.calc.TimeToBase
}

func (i *InclinedPlaneScene) initialHeight() float64 {
	if i.snapshot.hBlock > 0 {
		return i.snapshot.hBlock
	}
	thetaRad := i.theta * math.Pi / 180.0
	return i.length * math.Sin(thetaRad)
}

func (i *InclinedPlaneScene) distanceFromBase() float64 {
	if i.phase == "incline" || i.phase == "ready" {
		d := i.calc.DistanceToBase - i.simS
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func (i *InclinedPlaneScene) currentAcceleration() float64 {
	if !i.calc.Slides {
		return 0
	}
	if i.phase == "horizontal" {
		if i.calc.HorizontalDecel <= 0 {
			return 0
		}
		return -i.calc.HorizontalDecel
	}
	if i.phase == "stopped" {
		return 0
	}
	return i.calc.Acceleration
}

func (i *InclinedPlaneScene) drawInclinedPlane(screen *ebiten.Image, liveDataBottom float64) {
	if i.theta <= 0 || i.length <= 0 {
		return
	}

	thetaRad := i.theta * math.Pi / 180.0
	cosTheta := math.Cos(thetaRad)

	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	textDim := config.GlobalConfig.TextDimension

	// Area disponibile per la rappresentazione (triangolo + tratto orizzontale)
	leftMargin := sw * 0.03
	actionRight := sw * 0.95
	triBaseY := sh * 0.87
	maxTopY := liveDataBottom + textDim*0.2
	maxH := triBaseY - maxTopY
	if maxH < sh*0.18 {
		maxH = sh * 0.18
	}
	triX0 := leftMargin

	// Triangolo: vertice in alto a sinistra (triX0, triTopY) → apex
	//            angolo retto in basso a sinistra (triX0, triBaseY)
	//            angolo θ in basso a destra (triX1, triBaseY)
	inclineHorizontalMeters := i.length * cosTheta
	if inclineHorizontalMeters < 0.001 {
		inclineHorizontalMeters = 0.001
	}
	horizontalMeters := i.calc.HorizontalStopDist
	if horizontalMeters <= 0 {
		horizontalMeters = inclineHorizontalMeters * 1.4
	} else {
		// Lascia margine oltre il punto di arresto (es. 7m -> 10m)
		horizontalMeters *= 10.0 / 7.0
	}
	actionMeters := inclineHorizontalMeters + horizontalMeters
	if actionMeters < 0.001 {
		actionMeters = 0.001
	}

	pxPerMeter := (actionRight - leftMargin) / actionMeters
	triBaseW := inclineHorizontalMeters * pxPerMeter
	triH := triBaseW * math.Tan(thetaRad)
	if triH > maxH {
		scale := maxH / triH
		triH = maxH
		triBaseW *= scale
		pxPerMeter *= scale
	}
	triX1 := triX0 + triBaseW
	triTopY := triBaseY - triH

	// Linea del suolo
	vector.StrokeLine(screen,
		float32(triX0), float32(triBaseY)+2,
		float32(actionRight), float32(triBaseY)+2,
		3, color.RGBA{160, 140, 100, 200}, false)

	// Triangolo riempito con plane.jpg
	if i.planeImage != nil {
		imgW := float32(i.planeImage.Bounds().Dx())
		imgH := float32(i.planeImage.Bounds().Dy())
		vertices := []ebiten.Vertex{
			{DstX: float32(triX0), DstY: float32(triTopY), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			{DstX: float32(triX0), DstY: float32(triBaseY), SrcX: 0, SrcY: imgH, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			{DstX: float32(triX1), DstY: float32(triBaseY), SrcX: imgW, SrcY: imgH, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		}
		screen.DrawTriangles(vertices, []uint16{0, 1, 2}, i.planeImage, &ebiten.DrawTrianglesOptions{})
	} else if i.whiteImage != nil {
		vertices := []ebiten.Vertex{
			{DstX: float32(triX0), DstY: float32(triTopY), SrcX: 0, SrcY: 0, ColorR: 0.55, ColorG: 0.45, ColorB: 0.25, ColorA: 1},
			{DstX: float32(triX0), DstY: float32(triBaseY), SrcX: 0, SrcY: 0, ColorR: 0.55, ColorG: 0.45, ColorB: 0.25, ColorA: 1},
			{DstX: float32(triX1), DstY: float32(triBaseY), SrcX: 0, SrcY: 0, ColorR: 0.55, ColorG: 0.45, ColorB: 0.25, ColorA: 1},
		}
		screen.DrawTriangles(vertices, []uint16{0, 1, 2}, i.whiteImage, &ebiten.DrawTrianglesOptions{})
	}

	// Contorno del triangolo
	outlineColor := color.RGBA{210, 210, 210, 255}
	lw := float32(2.5)
	vector.StrokeLine(screen, float32(triX0), float32(triTopY), float32(triX0), float32(triBaseY), lw, outlineColor, false)
	vector.StrokeLine(screen, float32(triX0), float32(triBaseY), float32(triX1), float32(triBaseY), lw, outlineColor, false)
	vector.StrokeLine(screen, float32(triX1), float32(triBaseY), float32(triX0), float32(triTopY), lw, outlineColor, false)

	// Marcatore angolo retto in basso a sinistra
	sqSize := float32(math.Min(triBaseW, triH) * 0.035)
	if sqSize < 6 {
		sqSize = 6
	}
	sqColor := color.RGBA{200, 200, 200, 180}
	vector.StrokeLine(screen, float32(triX0)+sqSize, float32(triBaseY), float32(triX0)+sqSize, float32(triBaseY)-sqSize, 1.5, sqColor, false)
	vector.StrokeLine(screen, float32(triX0), float32(triBaseY)-sqSize, float32(triX0)+sqSize, float32(triBaseY)-sqSize, 1.5, sqColor, false)

	// Arco angolo θ in basso a destra
	arcRadius := float32(math.Min(triBaseW*0.12, 55))
	if arcRadius < 14 {
		arcRadius = 14
	}
	steps := 24
	for s := 0; s < steps; s++ {
		a0 := math.Pi - (thetaRad*float64(s))/float64(steps)
		a1 := math.Pi - (thetaRad*float64(s+1))/float64(steps)
		x0 := triX1 + float64(arcRadius)*math.Cos(a0)
		y0 := triBaseY - float64(arcRadius)*math.Sin(a0)
		x1 := triX1 + float64(arcRadius)*math.Cos(a1)
		y1 := triBaseY - float64(arcRadius)*math.Sin(a1)
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 2, color.RGBA{255, 220, 40, 230}, false)
	}

	// Etichetta θ
	labelX := triX0 + triBaseW*0.06
	labelY := triBaseY - float64(arcRadius)*1.15
	angleText := fmt.Sprintf("θ=%.1f°", i.theta)
	utils.ScreenDraw(-(textDim * 0.4), labelX+1.5, labelY+1.5, "black", screen, angleText, "libertinus")
	utils.ScreenDraw(-(textDim * 0.4), labelX, labelY, "cyan", screen, angleText, "libertinus")

	// Blocco
	if i.blockImage == nil {
		return
	}
	bImgW := float64(i.blockImage.Bounds().Dx())
	bImgH := float64(i.blockImage.Bounds().Dy())
	slopePixels := math.Sqrt(triBaseW*triBaseW + triH*triH)
	targetSide := math.Min(slopePixels*0.10, sh*0.095)
	if targetSide < slopePixels*0.07 {
		targetSide = slopePixels * 0.07
	}
	if targetSide < 10 {
		targetSide = 10
	}
	bScale := targetSide / math.Max(bImgW, bImgH)
	bW := bImgW * bScale
	bH := bImgH * bScale

	slopeLen := i.length
	distToBase := i.calc.DistanceToBase
	if distToBase > slopeLen {
		distToBase = slopeLen
	}

	if i.phase == "horizontal" || i.phase == "stopped" {
		// Blocco sul piano orizzontale dopo la base del piano inclinato
		gx := triX1 + i.simHorizS*pxPerMeter
		if gx > actionRight {
			gx = actionRight
		}
		contactInset := 1.5
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(bScale, bScale)
		op.GeoM.Translate(-bW, -bH)
		op.GeoM.Translate(gx, triBaseY+contactInset)
		screen.DrawImage(i.blockImage, op)
		return
	}

	// Blocco sul piano inclinato (fasi "ready" e "incline")
	// Riferimento: vertice in basso a destra (base del piano)
	distFromBase := distToBase - i.simS
	if distFromBase < 0 {
		distFromBase = 0
	}
	if distFromBase > slopeLen {
		distFromBase = slopeLen
	}
	dPix := distFromBase * pxPerMeter
	sxTouch := triX1 - dPix*math.Cos(thetaRad)
	syTouch := triBaseY - dPix*math.Sin(thetaRad)
	contactInset := 1.5
	sx := sxTouch - math.Sin(thetaRad)*contactInset
	sy := syTouch + math.Cos(thetaRad)*contactInset
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bScale, bScale)
	op.GeoM.Translate(-bW, -bH)
	op.GeoM.Rotate(thetaRad)
	op.GeoM.Translate(sx, sy)
	screen.DrawImage(i.blockImage, op)
}

var _ Scene = (*InclinedPlaneScene)(nil)

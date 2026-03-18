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

type projectileMotionSnapshot struct {
	// Snapshot dei valori di input usati per poter fare reset coerente.
	v0         float64
	thetaDeg   float64
	h0         float64
	rangeVal   float64
	timeVal    float64
	gravity    float64
	v0Set      bool
	thetaSet   bool
	hSet       bool
	rangeSet   bool
	timeSet    bool
	gravitySet bool
}

type ProjectileMotionScene struct {
	// calc contiene tutti i parametri risolti del problema.
	// simState rappresenta lo stato cinematico corrente nel tempo.
	calc     ProjectileMotionCalculus
	snapshot projectileMotionSnapshot
	simState ProjectileMotionSimState
	started  bool

	playImage  *ebiten.Image
	pauseImage *ebiten.Image
	arrowImage *ebiten.Image

	solveError string
}

func NewProjectileMotionScene() *ProjectileMotionScene {
	return &ProjectileMotionScene{}
}

func (p *ProjectileMotionScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return reason == Unpause
}

func (p *ProjectileMotionScene) Draw(screen *ebiten.Image) {
	screen.Clear()

	textDim := config.GlobalConfig.TextDimension
	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	leftX := sw * 0.03
	rightX := sw * 0.80
	y := textDim * 0.75
	step := textDim * 0.68
	smallSize := -(textDim * 0.54)

	// Stato testuale della simulazione mostrato nel pannello LIVE DATA.
	status := "REST - Press SPACE to start"
	if p.started && !p.simState.Completed {
		status = "RUNNING - Press SPACE to pause"
	}
	if p.simState.Completed {
		status = "COMPLETED - Press R to reset"
	}
	if p.solveError != "" {
		status = "INVALID INPUTS - Press R to reset"
	}

	// Dati in tempo reale (stato dinamico).
	leftLines := []string{
		"LIVE DATA",
		fmt.Sprintf("status: %s", status),
		fmt.Sprintf("t: %.2f s", p.simState.Time),
		fmt.Sprintf("x: %.2f m", p.simState.X),
		fmt.Sprintf("y: %.2f m", p.simState.Y),
		fmt.Sprintf("vx: %.2f m/s", p.simState.Vx),
		fmt.Sprintf("vy: %.2f m/s", p.simState.Vy),
		fmt.Sprintf("speed: %.2f m/s", p.simState.Speed),
		fmt.Sprintf("angle(v): %.2f\u00b0", p.simState.AngleDeg),
	}

	// Dati statici/risolti del problema (stato iniziale + risultati globali).
	rightLines := []string{
		"PROBLEM DATA",
		fmt.Sprintf("v0 (initial speed): %.2f m/s", p.calc.V0),
		fmt.Sprintf("\u03b8 (angle): %.2f\u00b0", p.calc.ThetaDeg),
		fmt.Sprintf("h (height): %.2f m", p.calc.H0),
		fmt.Sprintf("R (range): %.2f m", p.calc.Range),
		fmt.Sprintf("t (time): %.2f s", p.calc.FlightTime),
		fmt.Sprintf("g (gravity): %.2f m/s^2", p.calc.G),
		fmt.Sprintf("apex t: %.2f s", p.calc.ApexTime),
		fmt.Sprintf("apex x: %.2f m", p.calc.ApexX),
		fmt.Sprintf("apex y: %.2f m", p.calc.ApexY),
	}

	liveDataBottom := y + float64(len(leftLines))*step + textDim*0.25
	// Render fisico del moto nello spazio disponibile sotto i pannelli testuali.
	p.drawProjectileMotion(screen, liveDataBottom)

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

	if p.simState.Completed && p.solveError == "" {
		// Recap centrale a fine simulazione, simile allo stile inclined plane.
		resultY := textDim * 0.90
		resultGap := step * 0.95
		textSize := textDim + smallSize
		if textSize < 8 {
			textSize = 8
		}

		drawCentered := func(line, col string, yy float64) {
			w, _ := utils.MeasureTextWithSize(line, textSize, "libertinus")
			x := sw/2 - w/2
			utils.ScreenDraw(smallSize, x, yy, col, screen, line, "libertinus")
		}

		drawCentered(
			fmt.Sprintf("TOTAL TIME: %.3f s", p.calc.FlightTime),
			"green",
			resultY,
		)
		drawCentered(
			fmt.Sprintf("RANGE: %.3f m", p.calc.Range),
			"cyan",
			resultY+resultGap,
		)
		drawCentered(
			fmt.Sprintf("MAX HEIGHT: %.3f m", p.calc.ApexY),
			"orange",
			resultY+resultGap*2,
		)
	}

	if p.solveError != "" {
		errSize := textDim - (textDim / 5)
		errText := "Solver error: " + p.solveError
		utils.ScreenDraw(-(textDim / 5), utils.XCenteredWithFont(errText, errSize, "libertinus"), sh*0.83, "red", screen, errText, "libertinus")
	}

	controls := "SPACE: start/pause  <-/->: scrub timeline  R: reset  ENTER: menu"
	utils.ScreenDraw(smallSize, textDim/3, sh-textDim*0.9, "light gray", screen, controls, "libertinus")

	p.drawTimelineControls(screen)
}

func (p *ProjectileMotionScene) FirstLoad() {
	// Congela i dati correnti di configurazione per garantire reset ripetibili.
	p.snapshot = projectileMotionSnapshot{
		v0:         config.GlobalConfig.ProjectileV0,
		thetaDeg:   config.GlobalConfig.ProjectileTheta,
		h0:         config.GlobalConfig.ProjectileH,
		rangeVal:   config.GlobalConfig.ProjectileRange,
		timeVal:    config.GlobalConfig.ProjectileTime,
		gravity:    config.GlobalConfig.ProjectileGravity,
		v0Set:      config.GlobalConfig.ProjectileV0Set,
		thetaSet:   config.GlobalConfig.ProjectileThetaSet,
		hSet:       config.GlobalConfig.ProjectileHSet,
		rangeSet:   config.GlobalConfig.ProjectileRangeSet,
		timeSet:    config.GlobalConfig.ProjectileTimeSet,
		gravitySet: config.GlobalConfig.ProjectileGravitySet,
	}

	p.resetSimulationFromSnapshot()
	p.loadControlImages()
}

func (p *ProjectileMotionScene) OnEnter() {}
func (p *ProjectileMotionScene) OnExit()  {}

func (p *ProjectileMotionScene) Update() SceneId {
	// SPACE: play/pause; frecce/barra: scrub; R: reset; ENTER: pausa/menu.
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		p.toggleRunState()
	}

	p.handleKeyboardScrub()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.handleMouseControl()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		p.resetSimulationFromSnapshot()
	}

	p.stepSimulation()

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return PauseSceneId
	}

	return ProjectileMotionSceneId
}

func (p *ProjectileMotionScene) refreshCalculus() {
	// Riesegue la risoluzione completa del problema dai valori in config.
	calc, err := ComputeProjectileMotionCalculus(config.GlobalConfig)
	if err != nil {
		// In caso di errore invalida i dati runtime e mostra messaggio in scena.
		p.solveError = err.Error()
		p.calc = ProjectileMotionCalculus{}
		p.simState = ProjectileMotionSimState{}
		return
	}
	p.solveError = ""
	p.calc = calc
	p.simState = p.calc.ComputeStateAtTime(0)
}

func (p *ProjectileMotionScene) resetSimulationFromSnapshot() {
	// Ripristina esattamente i valori iniziali acquisiti in FirstLoad.
	config.GlobalConfig.ProjectileV0 = p.snapshot.v0
	config.GlobalConfig.ProjectileTheta = p.snapshot.thetaDeg
	config.GlobalConfig.ProjectileH = p.snapshot.h0
	config.GlobalConfig.ProjectileRange = p.snapshot.rangeVal
	config.GlobalConfig.ProjectileTime = p.snapshot.timeVal
	config.GlobalConfig.ProjectileGravity = p.snapshot.gravity
	config.GlobalConfig.ProjectileV0Set = p.snapshot.v0Set
	config.GlobalConfig.ProjectileThetaSet = p.snapshot.thetaSet
	config.GlobalConfig.ProjectileHSet = p.snapshot.hSet
	config.GlobalConfig.ProjectileRangeSet = p.snapshot.rangeSet
	config.GlobalConfig.ProjectileTimeSet = p.snapshot.timeSet
	config.GlobalConfig.ProjectileGravitySet = p.snapshot.gravitySet

	p.started = false
	p.refreshCalculus()
}

func (p *ProjectileMotionScene) stepSimulation() {
	// Avanza nel tempo con dt legato al TPS effettivo del motore.
	if p.solveError != "" {
		return
	}
	if !p.started || p.simState.Completed || p.calc.TotalDuration() <= 0 {
		return
	}

	tps := ebiten.ActualTPS()
	if tps <= 0 {
		tps = 60
	}
	dt := 1.0 / tps

	p.simState = p.calc.ComputeStateAtTime(p.simState.Time + dt)
	// Arresto automatico quando viene raggiunto il tempo di volo finale.
	if p.simState.Completed {
		p.started = false
	}
}

func (p *ProjectileMotionScene) toggleRunState() {
	// Non parte se i dati non sono risolti; se era finita, riparte da t=0.
	if p.solveError != "" || p.calc.TotalDuration() <= 0 {
		return
	}
	if p.simState.Completed {
		p.simState = p.calc.ComputeStateAtTime(0)
	}
	p.started = !p.started
}

func (p *ProjectileMotionScene) handleKeyboardScrub() {
	// Scrub temporale da tastiera (sinistra/destra) con passo adattivo.
	if p.solveError != "" {
		return
	}
	total := p.calc.TotalDuration()
	if total <= 0 {
		return
	}

	step := total * 0.02
	if step < 0.05 {
		step = 0.05
	}

	advance := func(key ebiten.Key, delta float64) {
		// Supporta anche pressione prolungata per uno scorrimento continuo.
		pressed := inpututil.IsKeyJustPressed(key)
		dur := inpututil.KeyPressDuration(key)
		if !pressed && dur > 10 && dur%3 == 0 {
			pressed = true
		}
		if !pressed {
			return
		}
		p.started = false
		p.simState = p.calc.ComputeStateAtTime(p.simState.Time + delta)
	}

	advance(ebiten.KeyArrowRight, step)
	advance(ebiten.KeyArrowLeft, -step)
}

func (p *ProjectileMotionScene) loadControlImages() {
	// Carica le icone UI (play/pause) e lo sprite del proiettile.
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

	p.playImage = load("img/play.png")
	p.pauseImage = load("img/pause.png")
	p.arrowImage = load("img/arrow.png")
}

func (p *ProjectileMotionScene) playButtonRect() (float64, float64, float64, float64) {
	textDim := config.GlobalConfig.TextDimension
	buttonSize := textDim * 1.25
	barX := float64(config.GlobalConfig.ScreenWidth) * 0.2
	barY := float64(config.GlobalConfig.ScreenHeight) - textDim*2.1
	buttonX := barX - buttonSize - textDim*0.45
	buttonY := barY - (buttonSize-textDim*0.5)/2
	return buttonX, buttonY, buttonSize, buttonSize
}

func (p *ProjectileMotionScene) progressBarRect() (float64, float64, float64, float64) {
	textDim := config.GlobalConfig.TextDimension
	barX := float64(config.GlobalConfig.ScreenWidth) * 0.2
	barW := float64(config.GlobalConfig.ScreenWidth) * 0.6
	barH := textDim * 0.45
	barY := float64(config.GlobalConfig.ScreenHeight) - textDim*2.0
	return barX, barY, barW, barH
}

func (p *ProjectileMotionScene) drawTimelineControls(screen *ebiten.Image) {
	// Disegna bottone play/pause, barra progresso e cursore temporale.
	btnX, btnY, btnW, btnH := p.playButtonRect()
	barX, barY, barW, barH := p.progressBarRect()

	vector.DrawFilledRect(screen, float32(btnX), float32(btnY), float32(btnW), float32(btnH), color.RGBA{40, 40, 40, 255}, false)

	icon := p.playImage
	if p.started && !p.simState.Completed {
		icon = p.pauseImage
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
	}

	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW), float32(barH), color.RGBA{55, 55, 55, 255}, false)
	progress := p.calc.SimProgress(p.simState.Time)
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW*progress), float32(barH), color.RGBA{30, 170, 90, 255}, false)

	knobX := barX + barW*progress
	if knobX < barX {
		knobX = barX
	}
	if knobX > barX+barW {
		knobX = barX + barW
	}
	vector.DrawFilledRect(screen, float32(knobX-2), float32(barY-4), 4, float32(barH+8), color.RGBA{230, 230, 230, 255}, false)
}

func (p *ProjectileMotionScene) handleMouseControl() {
	// Click sul bottone per play/pause o sulla barra per saltare a un istante.
	if p.solveError != "" {
		return
	}

	mx, my := ebiten.CursorPosition()
	px := float64(mx)
	py := float64(my)

	btnX, btnY, btnW, btnH := p.playButtonRect()
	if px >= btnX && px <= btnX+btnW && py >= btnY && py <= btnY+btnH {
		p.toggleRunState()
		return
	}

	barX, barY, barW, barH := p.progressBarRect()
	if px >= barX && px <= barX+barW && py >= barY && py <= barY+barH {
		total := p.calc.TotalDuration()
		if total <= 0 {
			return
		}
		wasRunning := p.started
		progress := (px - barX) / barW
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		p.simState = p.calc.ComputeStateAtTime(total * progress)
		p.started = wasRunning && !p.simState.Completed
	}
}

func (p *ProjectileMotionScene) drawProjectileMotion(screen *ebiten.Image, liveDataBottom float64) {
	// Rendering geometrico della traiettoria e dello sprite in base allo stato corrente.
	if p.solveError != "" || p.calc.TotalDuration() <= 0 {
		return
	}

	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	textDim := config.GlobalConfig.TextDimension

	left := sw * 0.08
	right := sw * 0.92
	top := liveDataBottom + textDim*0.2
	bottom := sh * 0.78

	if bottom-top < sh*0.2 {
		bottom = top + sh*0.2
	}

	// Bounding box logica (metri) della scena fisica, con piccolo margine visivo.
	maxX := math.Max(5.0, p.calc.Range*1.08)
	maxY := math.Max(2.0, math.Max(p.calc.ApexY, p.calc.H0)*1.10)

	scaleX := (right - left) / maxX
	scaleY := (bottom - top) / maxY
	scale := math.Min(scaleX, scaleY)
	if scale <= 0 {
		return
	}

	// Conversione coordinate fisiche (m) -> coordinate schermo (px).
	toScreen := func(x, y float64) (float64, float64) {
		sx := left + x*scale
		sy := bottom - y*scale
		return sx, sy
	}

	gx0, gy := toScreen(0, 0)
	gx1, _ := toScreen(maxX, 0)
	vector.StrokeLine(screen, float32(gx0), float32(gy), float32(gx1), float32(gy), 3, color.RGBA{180, 180, 180, 255}, false)

	curT := p.simState.Time
	total := p.calc.TotalDuration()
	if curT > total {
		curT = total
	}
	if curT < 0 {
		curT = 0
	}

	// Traccia bianca tratteggiata: campiona la curva fino al tempo corrente.
	samples := 140
	for s := 0; s < samples; s++ {
		t0 := curT * float64(s) / float64(samples)
		t1 := curT * float64(s+1) / float64(samples)
		st0 := p.calc.ComputeStateAtTime(t0)
		st1 := p.calc.ComputeStateAtTime(t1)
		x0, y0 := toScreen(st0.X, st0.Y)
		x1, y1 := toScreen(st1.X, st1.Y)
		if s%2 == 0 {
			vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 2, color.RGBA{240, 240, 240, 255}, false)
		}
	}

	// Marcatori di partenza e atterraggio.
	startX, startY := toScreen(0, p.calc.H0)
	vector.DrawFilledCircle(screen, float32(startX), float32(startY), 4, color.RGBA{60, 180, 255, 255}, false)
	landX, landY := toScreen(p.calc.Range, 0)
	vector.DrawFilledCircle(screen, float32(landX), float32(landY), 4, color.RGBA{130, 255, 130, 255}, false)

	// Posizione istantanea del proiettile.
	x, y := toScreen(p.simState.X, p.simState.Y)
	if p.arrowImage != nil {
		imgW := float64(p.arrowImage.Bounds().Dx())
		imgH := float64(p.arrowImage.Bounds().Dy())
		if imgW > 0 && imgH > 0 {
			target := math.Max(20, textDim*0.9)
			s := target / math.Max(imgW, imgH)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(s, s)
			dw := imgW * s
			dh := imgH * s
			op.GeoM.Translate(-dw/2, -dh/2)
			// Ruota la freccia lungo la direzione del vettore velocita.
			angleRad := math.Atan2(p.simState.Vy, p.simState.Vx)
			op.GeoM.Rotate(-angleRad)
			op.GeoM.Translate(x, y)
			screen.DrawImage(p.arrowImage, op)
			return
		}
	}
	vector.DrawFilledCircle(screen, float32(x), float32(y), 6, color.RGBA{255, 250, 180, 255}, false)
}

var _ Scene = (*ProjectileMotionScene)(nil)

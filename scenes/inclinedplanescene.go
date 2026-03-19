package scenes

import (
	"fmt"
	"image/color"
	"math"
	"physiGo/config"
	"physiGo/utils"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type inclinedPlaneSnapshot struct {
	objectMode string
	rotaryType string
	radius     float64

	theta      float64
	muS        float64
	muK        float64
	muR        float64
	mass       float64
	gravity    float64
	length     float64
	hBlock     float64
	v0         float64
	muSSet     bool
	muKSet     bool
	muRSet     bool
	gravitySet bool
}

type InclinedPlaneScene struct {
	calc     InclinedPlaneCalculus
	snapshot inclinedPlaneSnapshot
	simState InclinedPlaneSimState
	started  bool

	playImage   *ebiten.Image
	pauseImage  *ebiten.Image
	planeImage  *ebiten.Image
	blockImage  *ebiten.Image
	barrelImage *ebiten.Image
	whiteImage  *ebiten.Image
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
	if i.started && !i.simState.Completed {
		status = "RUNNING - Press SPACE to pause"
	}
	if i.simState.Completed {
		status = "COMPLETED - Press R to reset"
	}

	phaseLabel := phaseDisplayName(i.simState.Phase)
	frictionWork := i.currentFrictionWork(i.simState)

	leftLines := []string{
		"LIVE DATA",
		fmt.Sprintf("status: %s", status),
		fmt.Sprintf("Phase: %s", phaseLabel),
		fmt.Sprintf("t: %.2f s", i.simState.Time),
		fmt.Sprintf("v: %.2f m/s", i.simState.Velocity),
		fmt.Sprintf("a: %.2f m/s^2", i.calc.CurrentAcceleration(i.simState.Phase)),
		fmt.Sprintf("s (Inclined): %.2f m", i.simState.S),
		fmt.Sprintf("s (ground): %.2f m", i.simState.HorizS),
		fmt.Sprintf("s Tot: %.2f m", i.simState.S+i.simState.HorizS),
		fmt.Sprintf("h: %.2f m", i.simState.HBlock),
		"",
		fmt.Sprintf("K (trans): %.2f J", i.simState.KineticTranslational),
		fmt.Sprintf("K (rot): %.2f J", i.simState.KineticRotational),
		fmt.Sprintf("U: %.2f J", i.simState.PotentialEnergy),
		fmt.Sprintf("W friction: %.2f J", frictionWork),
		fmt.Sprintf("W total: %.2f J", i.simState.TotalWork),
	}

	bodyLabel := "blocco"
	isRotary := i.calc.ObjectMode == InclinedObjectRotary
	if isRotary {
		bodyLabel = "rotatorio"
	}

	muSLabel := "-"
	if i.snapshot.muSSet {
		muSLabel = fmt.Sprintf("%.2f", i.snapshot.muS)
	}
	muKLabel := "-"
	if i.snapshot.muKSet {
		muKLabel = fmt.Sprintf("%.2f", i.snapshot.muK)
	}
	muRLabel := "-"
	if i.snapshot.muRSet {
		muRLabel = fmt.Sprintf("%.2f", i.snapshot.muR)
	}

	rotaryShapeLabel := rotaryTypeLabel(i.calc.RotaryType)
	rotaryFormula := rotaryInertiaFormula(i.calc.RotaryType)

	fTotalBlock := i.calc.WeightParallel
	fAttrito := i.calc.DynamicFriction
	if isRotary {
		fAttrito = 0
	}

	rightLines := []string{}
	if isRotary {
		rollingForce := i.calc.Mass * i.calc.HorizontalDecel
		rightLines = []string{
			"PLANE DATA",
			fmt.Sprintf("Body: %s", bodyLabel),
			fmt.Sprintf("Shape: %s", rotaryShapeLabel),
			fmt.Sprintf("I formula: %s", rotaryFormula),
			fmt.Sprintf("Radius: %.2f m", i.calc.Radius),
			fmt.Sprintf("Inertia I: %.3f kg*m^2", i.calc.MomentOfInertia),
			fmt.Sprintf("Mass (m): %.2f kg", i.snapshot.mass),
			fmt.Sprintf("Gravity (g): %.2f m/s^2", i.snapshot.gravity),
			fmt.Sprintf("θ: %.1f°", i.snapshot.theta),
			fmt.Sprintf("Length (L): %.2f m", i.snapshot.length),
			fmt.Sprintf("Block Height (h): %.2f m", i.calc.InitialHeight),
			fmt.Sprintf("v0: %.2f m/s", i.snapshot.v0),
			fmt.Sprintf("μ_r: %s", muRLabel),
			"",
			fmt.Sprintf("F total (body): %.2f N", fTotalBlock),
			fmt.Sprintf("Fr (rolling): %.2f N", rollingForce),
			fmt.Sprintf("Ftot - Fr (net): %.2f N", i.calc.NetForce),
			fmt.Sprintf("Slides: %t", i.calc.Slides),
		}
	} else {
		rightLines = []string{
			"PLANE DATA",
			fmt.Sprintf("Body: %s", bodyLabel),
			fmt.Sprintf("Mass (m): %.2f kg", i.snapshot.mass),
			fmt.Sprintf("Gravity (g): %.2f m/s^2", i.snapshot.gravity),
			fmt.Sprintf("θ: %.1f°", i.snapshot.theta),
			fmt.Sprintf("Length (L): %.2f m", i.snapshot.length),
			fmt.Sprintf("Block Height (h): %.2f m", i.calc.InitialHeight),
			fmt.Sprintf("v0: %.2f m/s", i.snapshot.v0),
			fmt.Sprintf("μ_s: %s", muSLabel),
			fmt.Sprintf("μ_k: %s", muKLabel),
			"",
			fmt.Sprintf("F total (block): %.2f N", fTotalBlock),
			fmt.Sprintf("Fa (friction): %.2f N", fAttrito),
			fmt.Sprintf("Ftot - Fa (net): %.2f N", i.calc.NetForce),
			fmt.Sprintf("Slides: %t", i.calc.Slides),
		}
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
	}, 0, 14)

	baseState := InclinedPlaneSimState{}
	hasBaseState := i.calc.Slides && !i.calc.StopsOnIncline && i.calc.TimeToBase >= 0
	if hasBaseState && i.simState.BaseReached {
		baseState = i.calc.ComputeStateAtTime(i.calc.TimeToBase)
	}

	if !i.calc.Slides {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: "BLOCK DOES NOT MOVE",
			col:  "red",
		})
	} else {
		resultLines = append(resultLines, struct {
			text string
			col  string
		}{
			text: "INCLINED PHASE",
			col:  "yellow",
		})

		if hasBaseState && i.simState.BaseReached {
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "Values at base",
				col:  "cyan",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: fmt.Sprintf("t: %.3fs  v: %.3fm/s", i.calc.TimeToBase, i.calc.VelocityAtBase),
				col:  "white",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: fmt.Sprintf("a: %.3fm/s^2", i.calc.Acceleration),
				col:  "white",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: fmt.Sprintf("K: %.3fJ  W: %.3fJ", baseState.KineticTranslational+baseState.KineticRotational, baseState.TotalWork),
				col:  "white",
			})
		}

		showHorizontalPhase := i.simState.BaseReached && (i.simState.SimulationEnded || i.calc.HorizontalDecel <= 0)
		if showHorizontalPhase {
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "",
				col:  "white",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "HORIZONTAL PHASE",
				col:  "yellow",
			})
			if i.calc.HorizontalDecel > 0 {
				resultLines = append(resultLines, struct {
					text string
					col  string
				}{
					text: fmt.Sprintf("t: %.3fs", i.calc.HorizontalStopTime),
					col:  "white",
				})
				resultLines = append(resultLines, struct {
					text string
					col  string
				}{
					text: fmt.Sprintf("a: %.3fm/s^2", -i.calc.HorizontalDecel),
					col:  "white",
				})
				resultLines = append(resultLines, struct {
					text string
					col  string
				}{
					text: fmt.Sprintf("x: %.3fm", i.calc.HorizontalStopDist),
					col:  "white",
				})
			}
		}

		if i.simState.BaseReached && !i.simState.SimulationEnded && i.calc.HorizontalDecel <= 0 {
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "No horizontal friction",
				col:  "orange",
			})
		}

		if i.simState.SimulationEnded {
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "",
				col:  "white",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: "FINAL RECAP",
				col:  "green",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: fmt.Sprintf("t tot: %.3fs", i.simState.SimulationEndTime),
				col:  "white",
			})
			resultLines = append(resultLines, struct {
				text string
				col  string
			}{
				text: fmt.Sprintf("x tot: %.3fm", i.calc.DistanceToBase+i.simState.SimulationEndHorizS),
				col:  "white",
			})
		}
	}

	for idx, line := range resultLines {
		drawCentered(line.text, line.col, resultY+float64(idx)*resultGap)
	}

	controls := "SPACE: start/pause <-/-> timeline R: reset ENTER: menu"
	_, barY, _, barH := i.progressBarRect()
	controlsW, _ := utils.MeasureTextWithSize(controls, textSize, "libertinus")
	controlsX := sw/2 - controlsW/2
	controlsY := barY + barH + textDim*0.85
	if controlsY > sh-textDim*0.3 {
		controlsY = sh - textDim*0.3
	}
	utils.ScreenDraw(smallSize, controlsX, controlsY, "light gray", screen, controls, "libertinus")

	i.drawTimelineControls(screen)
}

// FirstLoad salva i dati iniziali (per poter resettare la simulazione) e prepara il modello fisico
func (i *InclinedPlaneScene) FirstLoad() {
	i.snapshot = inclinedPlaneSnapshot{
		objectMode: config.GlobalConfig.InclinedObjectMode,
		rotaryType: config.GlobalConfig.InclinedRotaryType,
		radius:     config.GlobalConfig.InclinedRadius,
		theta:      config.GlobalConfig.InclinedTheta,
		muS:        config.GlobalConfig.InclinedMuS,
		muK:        config.GlobalConfig.InclinedMuK,
		muR:        config.GlobalConfig.InclinedMuR,
		mass:       config.GlobalConfig.InclinedMass,
		gravity:    config.GlobalConfig.InclinedGravity,
		length:     config.GlobalConfig.InclinedLength,
		hBlock:     config.GlobalConfig.InclinedHBlock,
		v0:         config.GlobalConfig.InclinedInitialVelocity,
		muSSet:     config.GlobalConfig.InclinedMuSSet,
		muKSet:     config.GlobalConfig.InclinedMuKSet,
		muRSet:     config.GlobalConfig.InclinedMuRSet,
		gravitySet: config.GlobalConfig.InclinedGravitySet,
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

// handleKeyboardScrub permette di scorrere la timeline da tastiera usando le frecce sinistra/destra
func (i *InclinedPlaneScene) handleKeyboardScrub() {
	delta, ok := timelineScrubDelta(i.calc.TotalDuration())
	if !ok {
		return
	}
	i.started = false
	i.simState = i.calc.ComputeStateAtTime(i.simState.Time + delta)
}

// refreshCalculus ricostruisce tutti i valori derivati del modello fisico
func (i *InclinedPlaneScene) refreshCalculus() {
	i.calc = ComputeInclinedPlaneCalculus(config.GlobalConfig)
}

// resetSimulationFromSnapshot ripristina i parametri iniziali salvati, ricalcola il modello e riporta la simulazione al tempo t=0.
func (i *InclinedPlaneScene) resetSimulationFromSnapshot() {
	config.GlobalConfig.InclinedObjectMode = i.snapshot.objectMode
	config.GlobalConfig.InclinedRotaryType = i.snapshot.rotaryType
	config.GlobalConfig.InclinedRadius = i.snapshot.radius
	config.GlobalConfig.InclinedTheta = i.snapshot.theta
	config.GlobalConfig.InclinedMuS = i.snapshot.muS
	config.GlobalConfig.InclinedMuK = i.snapshot.muK
	config.GlobalConfig.InclinedMuR = i.snapshot.muR
	config.GlobalConfig.InclinedMass = i.snapshot.mass
	config.GlobalConfig.InclinedGravity = i.snapshot.gravity
	config.GlobalConfig.InclinedLength = i.snapshot.length
	config.GlobalConfig.InclinedHBlock = i.snapshot.hBlock
	config.GlobalConfig.InclinedInitialVelocity = i.snapshot.v0
	config.GlobalConfig.InclinedMuSSet = i.snapshot.muSSet
	config.GlobalConfig.InclinedMuKSet = i.snapshot.muKSet
	config.GlobalConfig.InclinedMuRSet = i.snapshot.muRSet
	config.GlobalConfig.InclinedGravitySet = i.snapshot.gravitySet

	i.started = false
	i.refreshCalculus()
	i.simState = i.calc.ComputeStateAtTime(0)
}

// stepSimulation avanza la simulazione di un frame usando il TPS reale. Se la simulazione termina, forza lo stato in pausa
func (i *InclinedPlaneScene) stepSimulation() {
	if !i.started || i.simState.Completed || !i.calc.Slides {
		return
	}

	tps := ebiten.ActualTPS()
	if tps <= 0 {
		tps = 60
	}
	dt := 1.0 / tps

	i.simState = i.calc.ComputeStateAtTime(i.simState.Time + dt)
	if i.simState.Completed {
		i.started = false
	}
}

// toggleRunState alterna play/pausa. Se la simulazione era completata, riparte da t=0.
func (i *InclinedPlaneScene) toggleRunState() {
	if !i.calc.Slides {
		return
	}
	if i.simState.Completed {
		i.simState = i.calc.ComputeStateAtTime(0)
	}
	i.started = !i.started
}

// loadControlImages carica tutte le immagini usate dai controlli e dalla scena
func (i *InclinedPlaneScene) loadControlImages() {
	i.playImage = loadImage("img/play.png")
	i.pauseImage = loadImage("img/pause.png")
	i.planeImage = loadImage("img/plane.jpg")
	i.blockImage = loadImage("img/block.png")
	i.barrelImage = loadImage("img/barrel.png")
	i.whiteImage = ebiten.NewImage(1, 1) // Crea una texture bianca da usare per il triangolo se plane.jpg non è disponibile
	i.whiteImage.Fill(color.White)
}

// playButtonRect restituisce il bounding box del pulsante play/pause.
func (i *InclinedPlaneScene) playButtonRect() (float64, float64, float64, float64) {
	button, _ := timelineRects()
	return button.x, button.y, button.w, button.h
}

// progressBarRect restituisce il bounding box della barra di avanzamento.
func (i *InclinedPlaneScene) progressBarRect() (float64, float64, float64, float64) {
	_, bar := timelineRects()
	return bar.x, bar.y, bar.w, bar.h
}

// drawTimelineControls disegna pulsante play/pause e progress bar sincronizzati con lo stato corrente della simulazione.
func (i *InclinedPlaneScene) drawTimelineControls(screen *ebiten.Image) {
	button, bar := timelineRects()
	running := i.started && !i.simState.Completed
	drawTimelineButton(screen, button, running, i.playImage, i.pauseImage, "PLAY")
	drawTimelineBar(screen, bar, i.calc.SimProgress(i.simState.Time), i.calc.BaseReachProgressFraction())
}

// handleMouseControl gestisce click su pulsante e barra timeline. Il click sulla barra posiziona direttamente il tempo simulato.
func (i *InclinedPlaneScene) handleMouseControl() {
	mx, my := ebiten.CursorPosition()
	px := float64(mx)
	py := float64(my)

	button, bar := timelineRects()
	if button.contains(px, py) {
		i.toggleRunState()
		return
	}

	if bar.contains(px, py) {
		total := i.calc.TotalDuration()
		if total <= 0 {
			return
		}
		wasRunning := i.started
		progress := progressFromCursorX(px, bar)
		i.simState = i.calc.ComputeStateAtTime(total * progress)
		i.started = wasRunning && !i.simState.Completed
	}
}

// drawInclinedPlane disegna la rappresentazione geometrica e dinamica: piano inclinato, angolo theta e corpo in movimento sulle due fasi.
func (i *InclinedPlaneScene) drawInclinedPlane(screen *ebiten.Image, liveDataBottom float64) {
	if i.calc.Theta <= 0 || i.calc.DistanceToBase <= 0 {
		return
	}

	thetaRad := i.calc.Theta * math.Pi / 180.0
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

	// Disegno del triangolo e calcolo delle dimensioni basate sui dati fisici (calcolo della base orizzontale e dell'altezza, con scaling per adattarsi all'area disponibile)
	inclineHorizontalMeters := i.calc.DistanceToBase * cosTheta
	if inclineHorizontalMeters < 0.001 {
		inclineHorizontalMeters = 0.001
	}
	horizontalMeters := i.calc.HorizontalStopDist
	if horizontalMeters <= 0 {
		horizontalMeters = inclineHorizontalMeters * 1.4
	} else {
		// Lascia del margine margine oltre il punto di arresto così che il piano non termini direttamente dove si ferma il corpo, ma si estenda un po' oltre
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
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 2.2, color.RGBA{245, 70, 70, 255}, false)
	}

	// Etichetta θ
	labelX := triX1 - float64(arcRadius)*1.6
	labelY := triBaseY - float64(arcRadius)*0.95
	angleText := fmt.Sprintf("θ=%.1f°", i.calc.Theta)
	utils.ScreenDraw(-(textDim * 0.4), labelX+1.5, labelY+1.5, "black", screen, angleText, "libertinus")
	utils.ScreenDraw(-(textDim * 0.4), labelX, labelY, "red", screen, angleText, "libertinus")

	// Body sprite (block or rotating barrel)
	bodyImage := i.blockImage
	rotateByPhysics := false
	if i.calc.ObjectMode == InclinedObjectRotary && i.barrelImage != nil {
		bodyImage = i.barrelImage
		rotateByPhysics = true
	}

	if bodyImage == nil {
		return
	}
	bImgW := float64(bodyImage.Bounds().Dx())
	bImgH := float64(bodyImage.Bounds().Dy())
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

	slopeLen := i.calc.DistanceToBase
	distToBase := i.calc.DistanceToBase
	if distToBase > slopeLen {
		distToBase = slopeLen
	}

	if i.simState.Phase == "horizontal" || i.simState.Phase == "stopped" {
		// Body sul piano orizzontale dopo la base del piano inclinato
		gx := triX1 + i.simState.HorizS*pxPerMeter
		contactInset := 1.5

		if rotateByPhysics {
			cx := gx - bW/2
			cy := triBaseY + contactInset - bH/2
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(bScale, bScale)
			op.GeoM.Translate(-bW/2, -bH/2)
			op.GeoM.Rotate(i.simState.RotationAngle)
			op.GeoM.Translate(cx, cy)
			screen.DrawImage(bodyImage, op)
			return
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(bScale, bScale)
		op.GeoM.Translate(-bW, -bH)
		op.GeoM.Translate(gx, triBaseY+contactInset)
		screen.DrawImage(bodyImage, op)
		return
	}

	// Blocco sul piano inclinato (fasi "ready" e "incline"), usando come riferimento il vertice in basso a destra (base del piano)
	distFromBase := distToBase - i.simState.S
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

	if rotateByPhysics {
		// Mantiene il pivot al centro e solleva lo sprite lungo la normale esterna al piano
		// per dare l'effetto di rotazione sul piano inclinato
		lift := math.Max(2.0, bH*0.07)
		cx := sx - bW/2 + math.Sin(thetaRad)*lift
		cy := sy - bH/2 - math.Cos(thetaRad)*lift
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(bScale, bScale)
		op.GeoM.Translate(-bW/2, -bH/2)
		op.GeoM.Rotate(i.simState.RotationAngle)
		op.GeoM.Translate(cx, cy)
		screen.DrawImage(bodyImage, op)
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bScale, bScale)
	op.GeoM.Translate(-bW, -bH)
	op.GeoM.Rotate(thetaRad)
	op.GeoM.Translate(sx, sy)
	screen.DrawImage(bodyImage, op)
}

// phaseDisplayName converte il nome interno della fase in una etichetta leggibile per la UI
func phaseDisplayName(phase string) string {
	switch phase {
	case "horizontal":
		return "Horizontal"
	case "stopped":
		return "Stopped"
	default:
		return "Inclined Plane"
	}
}

// currentFrictionForce restituisce il valore della forza d'attrito attiva nella fase corrente della simulazione
func (i *InclinedPlaneScene) currentFrictionForce(state InclinedPlaneSimState) float64 {
	if !i.calc.Slides {
		return 0
	}
	if state.Phase == "incline" || state.Phase == "ready" {
		if i.calc.ObjectMode == InclinedObjectRotary {
			return 0
		}
		return i.calc.DynamicFriction
	}
	if state.Phase == "horizontal" && i.calc.HorizontalDecel > 0 {
		return i.calc.Mass * i.calc.HorizontalDecel
	}
	return 0
}

// currentFrictionWork calcola il lavoro dell'attrito accumulato fino allo stato corrente, separando contributo su piano inclinato e orizzontale
func (i *InclinedPlaneScene) currentFrictionWork(state InclinedPlaneSimState) float64 {
	work := 0.0
	if i.calc.ObjectMode != InclinedObjectRotary && i.calc.DynamicFriction > 0 && state.S > 0 {
		work -= i.calc.DynamicFriction * state.S
	}
	if i.calc.HorizontalDecel > 0 && state.HorizS > 0 {
		work -= i.calc.Mass * i.calc.HorizontalDecel * state.HorizS
	}
	return work
}

var _ Scene = (*InclinedPlaneScene)(nil)

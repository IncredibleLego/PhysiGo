package scenes

import (
	"math"
	"physiGo/config"
	"physiGo/utils"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const maxProjectileAngle = 89.0

type ProjectileMotionInputScene struct {
	activeField     int
	overwriteOnType bool

	v0Input      string
	thetaInput   string
	hInput       string
	rangeInput   string
	timeInput    string
	gravityInput string

	lastBlink time.Time

	validationMessage string
}

func NewProjectileMotionInputScene() *ProjectileMotionInputScene {
	return &ProjectileMotionInputScene{}
}

func (p *ProjectileMotionInputScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return false
}

func (p *ProjectileMotionInputScene) Draw(screen *ebiten.Image) {
	screen.Clear()

	textDim := config.GlobalConfig.TextDimension
	spacing := textDim * 1.55
	startY := float64(config.GlobalConfig.ScreenHeight) * 0.23

	title := "PROJECTILE MOTION SETUP"
	utils.ScreenDraw(0, utils.XCenteredWithFont(title, textDim, "libertinus"), startY-textDim*1.2, "yellow", screen, title, "libertinus")

	// Costruisce le righe del form con etichetta, valore corrente e unita.
	lines := []string{
		"v₀ (initial speed): " + utils.RenderInputValue(&p.lastBlink, p.v0Input, 0, p.activeField, " m/s"),
		"\u03b8 (angle): " + utils.RenderInputValue(&p.lastBlink, p.thetaInput, 1, p.activeField, " \u00b0"),
		"h (height): " + utils.RenderInputValue(&p.lastBlink, p.hInput, 2, p.activeField, " m"),
		"R (range): " + utils.RenderInputValue(&p.lastBlink, p.rangeInput, 3, p.activeField, " m"),
		"t (time): " + utils.RenderInputValue(&p.lastBlink, p.timeInput, 4, p.activeField, " s"),
		"g (gravity): " + utils.RenderInputValue(&p.lastBlink, p.gravityInput, 5, p.activeField, " m/s²"),
	}

	for idx, line := range lines {
		col := "white"
		if idx == p.activeField {
			col = "cyan"
		}
		y := startY + float64(idx)*spacing
		utils.ScreenDraw(0, utils.XCenteredWithFont(line, textDim, "libertinus"), y, col, screen, line, "libertinus")
	}

	if p.validationMessage != "" {
		y := startY + float64(len(lines))*spacing + textDim*0.8
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(p.validationMessage, smallText, "libertinus"), y, "red", screen, p.validationMessage, "libertinus")
	}

	status := "Insert at least 2 non-zero values among v₀, \u03b8, h, R, t"
	if p.allInputsValid() {
		status = "Values ready - press Enter to continue"
	}
	y := startY + float64(len(lines))*spacing + textDim*2.1
	smallText := textDim - (textDim / 3)
	utils.ScreenDraw(-(textDim / 3), utils.XCenteredWithFont(status, smallText, "libertinus"), y, "light gray", screen, status, "libertinus")
}

func (p *ProjectileMotionInputScene) FirstLoad() {
	// Valori iniziali: g preimpostata, gli altri campi a 0.
	p.activeField = 0
	p.overwriteOnType = true
	p.v0Input = "0"
	p.thetaInput = "0"
	p.hInput = "0"
	p.rangeInput = "0"
	p.timeInput = "0"
	p.gravityInput = "9.8"
	p.lastBlink = time.Now()
	p.validationMessage = ""
}

func (p *ProjectileMotionInputScene) OnEnter() {}
func (p *ProjectileMotionInputScene) OnExit()  {}

func (p *ProjectileMotionInputScene) Update() SceneId {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return PauseSceneId
	}
	prevField := p.activeField

	// Navigazione verticale ciclica tra i campi del form.
	fieldCount := 6
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		p.activeField--
		if p.activeField < 0 {
			p.activeField = fieldCount - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		p.activeField++
		if p.activeField > fieldCount-1 {
			p.activeField = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		// Enter valida il campo attivo; sull'ultimo campo tenta il submit completo.
		if p.tryConfirmActiveField() {
			p.validationMessage = ""
			if p.activeField < fieldCount-1 {
				p.activeField++
				p.overwriteOnType = true
			} else if p.allInputsValid() {
				p.storeValues()
				return ProjectileMotionSceneId
			}
		}
	}

	if p.activeField != prevField {
		p.overwriteOnType = true
	}

	p.handleActiveFieldInput()
	return ProjectileMotionInputSceneId
}

func (p *ProjectileMotionInputScene) handleActiveFieldInput() {
	// Instrada l'input numerico verso la stringa del campo selezionato.
	switch p.activeField {
	case 0:
		p.handleNumericInput(&p.v0Input)
	case 1:
		p.handleNumericInput(&p.thetaInput)
	case 2:
		p.handleNumericInput(&p.hInput)
	case 3:
		p.handleNumericInput(&p.rangeInput)
	case 4:
		p.handleNumericInput(&p.timeInput)
	case 5:
		p.handleNumericInput(&p.gravityInput)
	}
}

func (p *ProjectileMotionInputScene) handleNumericInput(input *string) {
	// Parser minimale da tastiera: cifre, un separatore decimale e backspace.
	handleNumericTextInput(input, 8, &p.overwriteOnType)
}

func (p *ProjectileMotionInputScene) tryConfirmActiveField() bool {
	// Validazione locale del solo campo attivo (feedback immediato all'utente).
	switch p.activeField {
	case 0:
		_, ok, _ := parseOptionalNonNegative(p.v0Input)
		if !ok {
			p.validationMessage = "v₀ must be ≥ 0"
			return false
		}
	case 1:
		theta, ok, _ := parseOptionalNonNegative(p.thetaInput)
		if !ok || theta > maxProjectileAngle {
			p.validationMessage = "\u03b8 must be between 0 and 89"
			return false
		}
	case 2:
		_, ok, _ := parseOptionalNonNegative(p.hInput)
		if !ok {
			p.validationMessage = "h must be ≥ 0"
			return false
		}
	case 3:
		_, ok, _ := parseOptionalNonNegative(p.rangeInput)
		if !ok {
			p.validationMessage = "R must be ≥ 0"
			return false
		}
	case 4:
		_, ok, _ := parseOptionalNonNegative(p.timeInput)
		if !ok {
			p.validationMessage = "t must be ≥ 0"
			return false
		}
	case 5:
		_, ok, _ := parseOptionalMin(p.gravityInput, 0)
		if !ok {
			p.validationMessage = "g must be greater than 0"
			return false
		}
	}
	return true
}

func (p *ProjectileMotionInputScene) allInputsValid() bool {
	// Validazione globale: range numerici + regola "almeno 2 valori non nulli"
	// e controllo finale di consistenza fisica tramite il solver.
	v0, ok, _ := parseOptionalNonNegative(p.v0Input)
	if !ok {
		return false
	}
	theta, ok, _ := parseOptionalNonNegative(p.thetaInput)
	if !ok || theta > maxProjectileAngle {
		p.validationMessage = "\u03b8 must be between 0 and 89"
		return false
	}
	h, ok, _ := parseOptionalNonNegative(p.hInput)
	if !ok {
		p.validationMessage = "h must be ≥ 0"
		return false
	}
	rg, ok, _ := parseOptionalNonNegative(p.rangeInput)
	if !ok {
		p.validationMessage = "R must be ≥ 0"
		return false
	}
	tf, ok, _ := parseOptionalNonNegative(p.timeInput)
	if !ok {
		p.validationMessage = "t must be ≥ 0"
		return false
	}
	g, ok, gSet := parseOptionalMin(p.gravityInput, 0)
	if !ok {
		p.validationMessage = "g must be greater than 0"
		return false
	}
	if !gSet {
		g = 9.8
	}

	knownCount := 0
	if math.Abs(v0) > projectileEpsilon {
		knownCount++
	}
	if math.Abs(theta) > projectileEpsilon {
		knownCount++
	}
	if math.Abs(h) > projectileEpsilon {
		knownCount++
	}
	if math.Abs(rg) > projectileEpsilon {
		knownCount++
	}
	if math.Abs(tf) > projectileEpsilon {
		knownCount++
	}
	if knownCount < 2 {
		p.validationMessage = "Insert at least 2 non-zero values among v₀, \u03b8, h, R, t"
		return false
	}

	_, err := SolveProjectileMotion(v0, theta, h, rg, tf, g)
	if err != nil {
		// Ritorna a schermo il messaggio del solver (input incompatibili).
		p.validationMessage = err.Error()
		return false
	}
	return true
}

func (p *ProjectileMotionInputScene) storeValues() {
	// Salva i valori validati in config globale e aggiorna i flag "set".
	v0, _, _ := parseOptionalNonNegative(p.v0Input)
	theta, _, _ := parseOptionalNonNegative(p.thetaInput)
	h, _, _ := parseOptionalNonNegative(p.hInput)
	rg, _, _ := parseOptionalNonNegative(p.rangeInput)
	tf, _, _ := parseOptionalNonNegative(p.timeInput)
	g, _, gSet := parseOptionalMin(p.gravityInput, 0)
	if !gSet {
		g = 9.8
	}

	config.GlobalConfig.ProjectileV0 = v0
	config.GlobalConfig.ProjectileTheta = theta
	config.GlobalConfig.ProjectileH = h
	config.GlobalConfig.ProjectileRange = rg
	config.GlobalConfig.ProjectileTime = tf
	config.GlobalConfig.ProjectileGravity = g

	config.GlobalConfig.ProjectileV0Set = math.Abs(v0) > projectileEpsilon
	config.GlobalConfig.ProjectileThetaSet = math.Abs(theta) > projectileEpsilon
	config.GlobalConfig.ProjectileHSet = math.Abs(h) > projectileEpsilon
	config.GlobalConfig.ProjectileRangeSet = math.Abs(rg) > projectileEpsilon
	config.GlobalConfig.ProjectileTimeSet = math.Abs(tf) > projectileEpsilon
	config.GlobalConfig.ProjectileGravitySet = true
}

var _ Scene = (*ProjectileMotionInputScene)(nil)

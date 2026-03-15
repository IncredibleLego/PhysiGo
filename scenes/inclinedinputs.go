package scenes

import (
	"math"
	"physiGo/config"
	"physiGo/utils"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const maxInclinedAngle = 89.0

type InclinedInputScene struct {
	activeField int

	thetaInput   string
	muSInput     string
	muKInput     string
	massInput    string
	gravityInput string
	lengthInput  string
	hBlockInput  string
	v0Input      string

	lastBlink time.Time

	validationMessage string
}

func NewInclinedInputScene() *InclinedInputScene {
	return &InclinedInputScene{}
}

func (i *InclinedInputScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return false
}

func (i *InclinedInputScene) Draw(screen *ebiten.Image) {
	screen.Clear()

	textDim := config.GlobalConfig.TextDimension
	spacing := textDim * 1.5
	startY := float64(config.GlobalConfig.ScreenHeight) * 0.2

	title := "INCLINED PLANE SETUP"
	utils.ScreenDraw(0, utils.XCenteredWithFont(title, textDim, "libertinus"), startY-textDim*1.2, "yellow", screen, title, "libertinus")

	lines := []string{
		"θ (0°-89°): " + i.renderInputValue(i.thetaInput, 0),
		"μ_s (>=0, optional): " + i.renderInputValue(i.muSInput, 1),
		"μ_k (>=0, optional): " + i.renderInputValue(i.muKInput, 2),
		"m (mass > 0): " + i.renderInputValue(i.massInput, 3),
		"g (gravity): " + i.renderInputValue(i.gravityInput, 4),
		"L (length > 0, optional): " + i.renderInputValue(i.lengthInput, 5),
		"h_block (height, optional): " + i.renderInputValue(i.hBlockInput, 6),
		"v0 (initial speed, optional): " + i.renderInputValue(i.v0Input, 7),
	}

	for idx, line := range lines {
		color := "white"
		if idx == i.activeField {
			color = "cyan"
		}
		y := startY + float64(idx)*spacing
		utils.ScreenDraw(0, utils.XCenteredWithFont(line, textDim, "libertinus"), y, color, screen, line, "libertinus")
	}

	if i.validationMessage != "" {
		y := startY + float64(len(lines))*spacing + textDim
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(i.validationMessage, smallText, "libertinus"), y, "red", screen, i.validationMessage, "libertinus")
	}

	status := "Use arrows to move, Enter to confirm"
	if i.allInputsValid() {
		status = "Values ready - press Enter to continue"
	}
	y := startY + float64(len(lines))*spacing + textDim*2.2
	smallText := textDim - (textDim / 3)
	utils.ScreenDraw(-(textDim / 3), utils.XCenteredWithFont(status, smallText, "libertinus"), y, "light gray", screen, status, "libertinus")
}

func (i *InclinedInputScene) FirstLoad() {
	i.activeField = 0
	i.thetaInput = ""
	i.muSInput = ""
	i.muKInput = ""
	i.massInput = ""
	i.gravityInput = "9.8"
	i.lengthInput = ""
	i.hBlockInput = ""
	i.v0Input = ""
	i.lastBlink = time.Now()
	i.validationMessage = ""
}

func (i *InclinedInputScene) OnEnter() {
}

func (i *InclinedInputScene) OnExit() {
}

func (i *InclinedInputScene) Update() SceneId {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		i.activeField--
		if i.activeField < 0 {
			i.activeField = 7
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		i.activeField++
		if i.activeField > 7 {
			i.activeField = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if i.tryConfirmActiveField() {
			i.validationMessage = ""
			if i.activeField < 7 {
				i.activeField++
			} else if i.allInputsValid() {
				i.storeValues()
				return InclinedPlaneSceneId
			}
		}
	}

	i.handleActiveFieldInput()

	return InclinedInputSceneId
}

func (i *InclinedInputScene) renderInputValue(value string, fieldIndex int) string {
	if fieldIndex != i.activeField {
		if value == "" {
			return "-"
		}
		// Add unit if field has been filled
		unit := ""
		switch fieldIndex {
		case 0:
			unit = "°"
		case 3:
			unit = " kg"
		case 4:
			unit = " m/s^2"
		case 5:
			unit = " m"
		case 6:
			unit = " m"
		case 7:
			unit = " m/s"
		}
		return value + unit
	}

	blinkOn := time.Since(i.lastBlink) < time.Second
	if time.Since(i.lastBlink) > time.Second*2 {
		i.lastBlink = time.Now()
	}
	if blinkOn {
		return value + "_"
	}
	if value == "" {
		return "-"
	}
	return value
}

func (i *InclinedInputScene) handleActiveFieldInput() {
	switch i.activeField {
	case 0:
		i.handleNumericInput(&i.thetaInput)
	case 1:
		i.handleNumericInput(&i.muSInput)
	case 2:
		i.handleNumericInput(&i.muKInput)
	case 3:
		i.handleNumericInput(&i.massInput)
	case 4:
		i.handleNumericInput(&i.gravityInput)
	case 5:
		i.handleNumericInput(&i.lengthInput)
	case 6:
		i.handleNumericInput(&i.hBlockInput)
	case 7:
		i.handleNumericInput(&i.v0Input)
	}
}

func (i *InclinedInputScene) handleNumericInput(input *string) {
	text := *input
	maxChars := 8

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(text) > 0 {
		text = text[:len(text)-1]
	}

	for key := ebiten.Key0; key <= ebiten.Key9; key++ {
		if inpututil.IsKeyJustPressed(key) && len(text) < maxChars {
			text += string('0' + rune(key-ebiten.Key0))
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) || inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		if !strings.ContainsAny(text, ".,") && len(text) < maxChars {
			if text == "" {
				text = "0"
			}
			text += "."
		}
	}

	*input = text
}

func (i *InclinedInputScene) tryConfirmActiveField() bool {
	switch i.activeField {
	case 0:
		_, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if !ok {
			i.validationMessage = "θ must be between 0 and 89"
			return false
		}
	case 1:
		_, ok, _ := parseOptionalNonNegative(i.muSInput)
		if !ok {
			i.validationMessage = "μ_s must be >= 0"
			return false
		}
	case 2:
		_, ok, _ := parseOptionalNonNegative(i.muKInput)
		if !ok {
			i.validationMessage = "μ_k must be >= 0"
			return false
		}
	case 3:
		_, ok, _ := parseOptionalMin(i.massInput, 0)
		if !ok {
			i.validationMessage = "m must be greater than 0"
			return false
		}
	case 4:
		_, ok, _ := parseOptionalMin(i.gravityInput, 0)
		if !ok {
			i.validationMessage = "g must be greater than 0"
			return false
		}
	case 5:
		_, ok, _ := parseOptionalMin(i.lengthInput, 0)
		if !ok {
			i.validationMessage = "L must be greater than 0"
			return false
		}
	case 6:
		if !i.validateHBlock() {
			return false
		}
	case 7:
		_, ok, _ := parseOptionalNonNegative(i.v0Input)
		if !ok {
			i.validationMessage = "v0 must be >= 0"
			return false
		}
	}

	return true
}

func (i *InclinedInputScene) allInputsValid() bool {
	if _, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.massInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.muSInput); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.muKInput); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.gravityInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.lengthInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.v0Input); !ok {
		return false
	}

	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) == "" {
		i.validationMessage = "Insert L or h_block"
		return false
	}

	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) != "" {
		theta, _ := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if theta <= 0 {
			i.validationMessage = "θ must be > 0 if only h_block is used"
			return false
		}
	}

	if !i.validateHBlock() {
		return false
	}
	return true
}

func (i *InclinedInputScene) storeValues() {
	theta, _ := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
	mass, _, massSet := parseOptionalMin(i.massInput, 0)
	length, _, lengthSet := parseOptionalMin(i.lengthInput, 0)
	muS, _, muSSet := parseOptionalNonNegative(i.muSInput)
	muK, _, muKSet := parseOptionalNonNegative(i.muKInput)
	gravity, _, gravitySet := parseOptionalMin(i.gravityInput, 0)
	v0, _, _ := parseOptionalNonNegative(i.v0Input)
	if !gravitySet {
		gravity = 9.8
	}
	if !massSet {
		mass = 1.0
	}

	// h_block is optional but defaults to 0 if not set.
	hBlock := 0.0
	if strings.TrimSpace(i.hBlockInput) != "" {
		hBlock, _ = parseFloatInput(i.hBlockInput)
	}

	if !lengthSet && hBlock > 0 {
		thetaRad := theta * math.Pi / 180.0
		sinTheta := math.Sin(thetaRad)
		if sinTheta > 0 {
			length = hBlock / sinTheta
		}
	}

	config.GlobalConfig.InclinedTheta = theta
	config.GlobalConfig.InclinedMass = mass
	config.GlobalConfig.InclinedLength = length
	config.GlobalConfig.InclinedHBlock = hBlock
	config.GlobalConfig.InclinedMuS = muS
	config.GlobalConfig.InclinedMuK = muK
	config.GlobalConfig.InclinedGravity = gravity
	config.GlobalConfig.InclinedInitialVelocity = v0
	config.GlobalConfig.InclinedMuSSet = muSSet
	config.GlobalConfig.InclinedMuKSet = muKSet
	config.GlobalConfig.InclinedGravitySet = gravitySet
}

func parseRequiredRange(input string, min, max float64) (float64, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false
	}
	if value < min || value > max {
		return 0, false
	}
	return value, true
}

func parseRequiredMin(input string, min float64) (float64, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false
	}
	if value <= min {
		return 0, false
	}
	return value, true
}

func parseOptionalMin(input string, min float64) (float64, bool, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, true, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false, true
	}
	if value <= min {
		return 0, false, true
	}
	return value, true, true
}

func parseOptionalNonNegative(input string) (float64, bool, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, true, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false, true
	}
	if value < 0 {
		return 0, false, true
	}
	return value, true, true
}

func parseFloatInput(input string) (float64, error) {
	clean := strings.ReplaceAll(strings.TrimSpace(input), ",", ".")
	return strconv.ParseFloat(clean, 64)
}

func (i *InclinedInputScene) validateHBlock() bool {
	// h_block can be empty (optional)
	if strings.TrimSpace(i.hBlockInput) == "" {
		return true
	}

	// Parse h_block
	hBlock, err := parseFloatInput(i.hBlockInput)
	if err != nil || hBlock <= 0 {
		i.validationMessage = "h_block must be greater than 0"
		return false
	}

	// Theta is always required for h-based geometry.
	thetaStr := strings.TrimSpace(i.thetaInput)
	lengthStr := strings.TrimSpace(i.lengthInput)

	if thetaStr == "" {
		i.validationMessage = "Enter theta first"
		return false
	}

	theta, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
	if !ok {
		i.validationMessage = "Enter valid theta first"
		return false
	}

	if lengthStr == "" {
		// If L is empty, it will be derived from h and theta.
		if theta <= 0 {
			i.validationMessage = "theta must be > 0 if L is empty"
			return false
		}
		return true
	}

	length, ok := parseRequiredMin(i.lengthInput, 0)
	if !ok {
		i.validationMessage = "Enter valid L first"
		return false
	}

	// Calculate max height: h_max = L * sin(theta)
	thetaRad := theta * math.Pi / 180.0
	hMax := length * math.Sin(thetaRad)

	// Ho usato epsilon perché a volte, a causa di arrotondamenti, hBlock potrebbe essere leggermente maggiore di hMax anche se l'utente ha inserito un valore corretto.
	// L'epsilon permette un piccolo margine di errore per evitare messaggi di validazione ingiusti.
	const epsilon = 0.01
	if hBlock > hMax+epsilon {
		i.validationMessage = "h_block must be <= L*sin(θ) = " + strconv.FormatFloat(hMax, 'f', 2, 64) + " m"
		return false
	}

	return true
}

var _ Scene = (*InclinedInputScene)(nil)

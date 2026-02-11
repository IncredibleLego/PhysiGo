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

type InclinedInputScene struct {
	activeField int

	thetaInput   string
	muSInput     string
	muKInput     string
	massInput    string
	gravityInput string
	lengthInput  string
	hBlockInput  string

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

	utils.ScreenDraw(0, utils.XCentered("INCLINED PLANE SETUP", textDim), startY-textDim*1.2, "yellow", screen, "INCLINED PLANE SETUP")

	lines := []string{
		"θ (0°-60°): " + i.renderInputValue(i.thetaInput, 0),
		"μ_s (0-1) (optional): " + i.renderInputValue(i.muSInput, 1),
		"μ_k (0-1) (optional): " + i.renderInputValue(i.muKInput, 2),
		"m (mass > 0): " + i.renderInputValue(i.massInput, 3),
		"g (gravity): " + i.renderInputValue(i.gravityInput, 4),
		"L (length > 0): " + i.renderInputValue(i.lengthInput, 5),
		"h_block (height): " + i.renderInputValue(i.hBlockInput, 6),
	}

	for idx, line := range lines {
		color := "white"
		if idx == i.activeField {
			color = "cyan"
		}
		y := startY + float64(idx)*spacing
		utils.ScreenDraw(0, utils.XCentered(line, textDim), y, color, screen, line)
	}

	if i.validationMessage != "" {
		y := startY + float64(len(lines))*spacing + textDim
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCentered(i.validationMessage, smallText), y, "red", screen, i.validationMessage)
	}

	status := "Use arrows to move, Enter to confirm"
	if i.allInputsValid() {
		status = "Values ready - press Enter to continue"
	}
	y := startY + float64(len(lines))*spacing + textDim*2.2
	smallText := textDim - (textDim / 3)
	utils.ScreenDraw(-(textDim / 3), utils.XCentered(status, smallText), y, "light gray", screen, status)
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
			i.activeField = 6
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		i.activeField++
		if i.activeField > 6 {
			i.activeField = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if i.tryConfirmActiveField() {
			i.validationMessage = ""
			if i.activeField < 6 {
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
		_, ok := parseRequiredRange(i.thetaInput, 0, 60)
		if !ok {
			i.validationMessage = "θ must be between 0 and 60"
			return false
		}
	case 1:
		_, ok, _ := parseOptionalRange(i.muSInput, 0, 1)
		if !ok {
			i.validationMessage = "μ_s must be between 0 and 1"
			return false
		}
	case 2:
		_, ok, _ := parseOptionalRange(i.muKInput, 0, 1)
		if !ok {
			i.validationMessage = "μ_k must be between 0 and 1"
			return false
		}
	case 3:
		_, ok := parseRequiredMin(i.massInput, 0)
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
		_, ok := parseRequiredMin(i.lengthInput, 0)
		if !ok {
			i.validationMessage = "L must be greater than 0"
			return false
		}
	case 6:
		if !i.validateHBlock() {
			return false
		}
	}

	return true
}

func (i *InclinedInputScene) allInputsValid() bool {
	if _, ok := parseRequiredRange(i.thetaInput, 0, 60); !ok {
		return false
	}
	if _, ok := parseRequiredMin(i.massInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalRange(i.muSInput, 0, 1); !ok {
		return false
	}
	if _, ok, _ := parseOptionalRange(i.muKInput, 0, 1); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.gravityInput, 0); !ok {
		return false
	}
	if _, ok := parseRequiredMin(i.lengthInput, 0); !ok {
		return false
	}
	if !i.validateHBlock() {
		return false
	}
	return true
}

func (i *InclinedInputScene) storeValues() {
	theta, _ := parseRequiredRange(i.thetaInput, 0, 60)
	mass, _ := parseRequiredMin(i.massInput, 0)
	length, _ := parseRequiredMin(i.lengthInput, 0)
	muS, _, muSSet := parseOptionalRange(i.muSInput, 0, 1)
	muK, _, muKSet := parseOptionalRange(i.muKInput, 0, 1)
	gravity, _, gravitySet := parseOptionalMin(i.gravityInput, 0)
	if !gravitySet {
		gravity = 9.8
	}

	// h_block is optional but defaults to 0 if not set
	hBlock := 0.0
	if strings.TrimSpace(i.hBlockInput) != "" {
		hBlock, _ = parseFloatInput(i.hBlockInput)
	}

	config.GlobalConfig.InclinedTheta = theta
	config.GlobalConfig.InclinedMass = mass
	config.GlobalConfig.InclinedLength = length
	config.GlobalConfig.InclinedHBlock = hBlock
	config.GlobalConfig.InclinedMuS = muS
	config.GlobalConfig.InclinedMuK = muK
	config.GlobalConfig.InclinedGravity = gravity
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

func parseOptionalRange(input string, min, max float64) (float64, bool, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, true, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false, true
	}
	if value < min || value > max {
		return 0, false, true
	}
	return value, true, true
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

	// Check if theta and length are available
	thetaStr := strings.TrimSpace(i.thetaInput)
	lengthStr := strings.TrimSpace(i.lengthInput)

	if thetaStr == "" || lengthStr == "" {
		i.validationMessage = "Enter theta and L first"
		return false
	}

	theta, ok := parseRequiredRange(i.thetaInput, 0, 60)
	if !ok {
		i.validationMessage = "Enter valid theta first"
		return false
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
		i.validationMessage = "h_block must be <= L*sin(theta) = " + strconv.FormatFloat(hMax, 'f', 2, 64) + " m"
		return false
	}

	return true
}

var _ Scene = (*InclinedInputScene)(nil)
